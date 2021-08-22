package main

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	dbg "runtime/debug"
	"strings"
	"time"

	"github.com/Alex-Eftimie/netutils"
	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
	"github.com/soheilhy/cmux"
)

var BWNoDomain []byte = socksDom("no-more-bandwidth.com")
var Socks5StatusBadGateway []byte = []byte{0x05, 0x04, 0x00, 0x01, 127, 0, 0, 1, 0x00, 0x00}
var Socks5StatusNoBW []byte = append(append([]byte{0x05, 0x02, 0x00, 0x03}, BWNoDomain...), []byte{0x00, 0x00}...)
var Socks5StatusNotAllowed []byte = []byte{0x05, 0x02, 0x00, 0x01, 127, 0, 0, 1, 0x00, 0x00}
var Socks5StatusSucceeded []byte = []byte{0x05, 0x00, 0x00, 0x01, 127, 0, 0, 1, 0x00, 0x00}

type socks5Server struct {
	AuthType string
	parent   *Server
}

func Socks5Matcher() cmux.Matcher {
	return func(r io.Reader) bool {
		b := make([]byte, 1)
		r.Read(b)
		return b[0] == 0x05
	}
}
func (ss *socks5Server) Close() {
	// TODO: Close server connections
}

func (ss *socks5Server) Serve(list net.Listener) {

	debug(999, fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "Listening on", list.Addr())

	for {
		rw, err := list.Accept()

		if err != nil {
			return
		}
		debug(999, fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "New connection from", rw.RemoteAddr())

		cc := &netutils.CounterConn{rw, 0, 0}
		if Co.DebugLevel > 9999 {
			cp := &PrinterConn{rw}
			cc = &netutils.CounterConn{cp, 0, 0}
		}
		go ss.ServeSocks(cc)
	}
}

func (ss *socks5Server) ServeSocks(c net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			dbg.PrintStack()
			fmt.Println(fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "Recovered from connection failed in f", r)
		}
	}()
	// read socks ver
	b1 := make([]byte, 1)
	n, err := c.Read(b1)
	if n != 1 || err != nil || b1[0] != 0x05 {
		log.Println(fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "e: socksver")
		c.Close()
		return
	}

	// read num methods
	b1 = make([]byte, 1)
	n, err = c.Read(b1)
	if n != 1 || err != nil {
		log.Println(fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "e: numm n")
		c.Close()
		return
	}

	num := int(b1[0])
	methodsb := make([]byte, num)

	n, err = c.Read(methodsb)

	// error reading num methods
	if err != nil || n != num {
		log.Println(fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "e: numm")
		return
	}

	var uinfo *netutils.UserInfo = nil
	m := make(map[string]string)
	// spew.Dump(methodsb, 0x02, ss.AuthType, num, b1)
	// log.Println("upass")
	methods := fmt.Sprintf("%d methods:", num)
	hasMethod := false
	for i := 0; i < n; i++ {
		methods = fmt.Sprintf("%s %d", methods, methodsb[i])
		// user & pass auth
		if methodsb[i] == 0x02 {
			hasMethod = true
		}
	}

	if !hasMethod {
		// if user/pass, then it's mandatory
		if ss.AuthType == AuthTypeUserPass {
			log.Printf("[Socks](%s) e: No upass auth: %s", ss.parent.Addr, methods)
			// no acceptable methods
			c.Write([]byte{0x05, 0xff})
			c.Close()
			return
		}
		debug(999, fmt.Sprintf("[Socks](%s) Proceeding with IP auth", ss.parent.Addr))
		// proceed with no-auth aka password
		c.Write([]byte{0x05, 0x00})

		// empty
		uinfo = &netutils.UserInfo{}
	} else {
		// auth pass
		c.Write([]byte{0x05, 0x02})
		uinfo, err = SocksPassAuth(c)
		if uinfo == nil || err != nil {
			log.Println(fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "Failed to get authentication:", err)
			c.Write([]byte{0x05, 0xff})
			c.Close()
			return
		}
	}

	// spew.Dump("uinfoooooooooooooooooooo", uinfo)
	if uinfo.User != "" {
		r := ParseParams(uinfo.User, &m)
		if r != "" {
			uinfo.User = r
		}
	}

	if uinfo.Pass != "" {
		r := ParseParams(uinfo.Pass, &m)
		// spew.Dump("XXXXXX", uinfo.Pass, r)
		if r != "" {
			uinfo.Pass = r
		}
	}

	if ss.AuthType == AuthTypeUserPass {
		if CheckUser(uinfo, ss.parent.Auth) == false {
			debug(99, fmt.Sprintf("[Socks](%s)", ss.parent.Addr),
				"Wrong User/Pass Combination:",
				"\n\tActual:", uinfo.User, uinfo.Pass,
				"\n\tExpected", ss.parent.Auth.User, ss.parent.Auth.Pass)

			c.Write([]byte{0x05, 0xff})
			c.Close()
			return
		}

	} else {
		ip := strings.Split(c.RemoteAddr().String(), ":")[0]

		if val, ok := ss.parent.Auth.IP[ip]; !ok || val == false {
			debug(999, fmt.Sprintf("[Socks](%s) IP not allowed: %s", ss.parent.Addr, c.RemoteAddr().String()))
			// no acceptable methods
			c.Write([]byte{0x05, 0xff})
			c.Close()
			return
		}
	}

	debug(999, fmt.Sprintf("[Socks](%s) Authentication passed, checking MaxThreads", ss.parent.Addr))
	if ss.parent.limiter.Add() == false {
		// no acceptable methods
		c.Write([]byte{0x05, 0xff})
		c.Close()
		return
	}
	defer ss.parent.limiter.Done()

	// if we're doing password authentication, then we need to confirm password
	if hasMethod {
		c.Write([]byte{0x05, 0x00})
	}

	var proxyP *ProxyInfo

	m["Group"] = "Default"

	// parses only the group to select the right proxy
	ParseProxyParams(&m, true)

	m["Group"] = strings.ToUpper(m["Group"])

	if ss.parent.Devices != nil {
		// if requested specific device
		d, ok := m["device"]
		if !ok {
			for v := range ss.parent.Devices {
				d = v
				break
			}
			m["device"] = d
		}
		proxyP = getDynamicProxy(d)
	} else if v, ok := m["sticky"]; ok { // IF sticky ip requested, get the same proxy
		key := uinfo.User + "::" + v

		// log.Println("get", key)
		proxyP = SessionMaster.GetSession(key)
		if proxyP == nil {
			// log.Println("set", key)
			proxyP = GetRandomProxy(m["Group"])
			SessionMaster.SetSession(key, proxyP)
		}
	} else {
		// log.Println("random proxy")
		proxyP = GetRandomProxy(m["Group"])
	}

	if proxyP == nil {
		c.Close()
		return
	}

	m["User"] = proxyP.User
	m["Pass"] = proxyP.Pass
	// parse the rest too
	ParseProxyParams(&m, false)

	proxy := &ProxyInfo{
		Host: proxyP.Host,
		Port: proxyP.Port,
		User: m["User"],
		Pass: m["Pass"],
		Type: proxyP.Type,
	}

	debug(999, fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "Handshake Success, waiting for stream:", c.RemoteAddr().String())

	ss.ServeConn(c, proxy)

	// if proxy.Type == TypeSocks5 {
	// 	destConn = forwardSocks(proxy, r, clientConn, done)
	// } else {
	// 	destConn = forwardHTTP(proxy, r, clientConn, done)
	// }
	// spew.Dump(uinfo, m, proxy)

	// forwarding

	// bw counting
}

type SocksRequestHeader struct {
	Bin struct {
		ver  byte
		cmd  byte
		rsv  byte
		atyp byte
		addr *[]byte
		port []byte
	}

	Port int
	Addr string
}

func (sr *SocksRequestHeader) WriteTo(c io.Writer) (int, error) {
	b := &sr.Bin
	data := []byte{b.ver, b.cmd, b.rsv, b.atyp}
	data = append(data, *b.addr...)
	data = append(data, b.port...)
	return c.Write(data)
}

func (ss *socks5Server) ServeConn(c net.Conn, proxy *ProxyInfo) {
	rh, err := ReadReqHeader(c)
	if rh == nil || err != nil {
		log.Println(fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "Failed to ReadReqHeader", err)
		c.Close()
		return
	}

	addr := net.JoinHostPort(rh.Addr, fmt.Sprintf("%d", rh.Port))

	if ss.parent.ExpireAt != nil && ss.parent.ExpireAt.Before(time.Now()) {
		debug(99, fmt.Sprintf("[Socks](%s) Expired: %s", ss.parent.Addr, c.RemoteAddr().String()))

		c.Write(Socks5StatusNotAllowed)
		c.Close()
		return
	}

	// close connection after the stream request if the server has no bandwidth remaining
	if !ss.parent.HasBW() {
		debug(99, fmt.Sprintf("[Socks](%s)", ss.parent.Addr), ss.parent.Addr, "Low Bandwidth")
		c.Write(Socks5StatusNoBW)
		c.Close()
		return
	}

	debug(999, fmt.Sprintf("[Socks](%s)", ss.parent.Addr), "Connect to proxy", proxy.Addr(), "from:", c.RemoteAddr().String())
	// color.Yellow("Connecting to %s", *addr)

	dc, err := net.DialTimeout("tcp", proxy.Addr(), 10*time.Second)
	if err != nil {
		color.Red("%s Error Connecting to %s", fmt.Sprintf("[Socks](%s)", ss.parent.Addr), addr)
		c.Write(Socks5StatusNoBW)
		c.Close()
		return
	}
	destConn := &netutils.CounterConn{dc, 0, 0}

	// Count all bytes transfered no matter what
	// perform connection logging
	defer func() {
		cl := c.(*netutils.CounterConn)

		up := destConn
		// log.Println("Done", s.Addr, dd.Downstream, dd.Upstream)

		if up.Upstream+up.Downstream > cl.Upstream+cl.Downstream {
			ss.parent.LogConnection(rh.Addr, "SOCKS5", up.Downstream, up.Upstream)
			ss.parent.Consume(up.Downstream, up.Upstream)
		} else {
			ss.parent.LogConnection(rh.Addr, "SOCKS5", cl.Downstream, cl.Upstream)
			ss.parent.Consume(cl.Downstream, cl.Upstream)
		}

	}()

	url := &url.URL{
		Host: addr,
	}
	nr := &http.Request{
		Method: "CONNECT",
		URL:    url,
		Header: make(map[string][]string),
	}

	if len(proxy.User) != 0 || len(proxy.Pass) != 0 {
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(proxy.Auth()))
		nr.Header.Set("Proxy-Authorization", basicAuth)
	} else {
		nr.Header.Del("Proxy-Authorization")
	}

	reader := bufio.NewReader(destConn)

	// Write the Request to the Upstream Proxy
	err = nr.WriteProxy(destConn)
	if err != nil {
		log.Println(fmt.Sprintf("[Socks](%s)", ss.parent.Addr), color.RedString("Proxy connection[WriteProxy] failed to "), proxy.Addr())
		c.Write(Socks5StatusBadGateway)
		c.Close()
		return
	}

	resp, err := http.ReadResponse(reader, nr)

	if err != nil || resp.StatusCode != 200 {
		c.Write(Socks5StatusBadGateway)
		c.Close()
		if resp != nil {
			log.Println(fmt.Sprintf("[Socks](%s)", ss.parent.Addr), color.RedString("Upstream error"), resp.StatusCode, resp.Status)
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			spew.Dump(resp.Header, string(bodyBytes))
			return
		}
		log.Println(fmt.Sprintf("[Socks](%s)", ss.parent.Addr), color.RedString("Proxy connection[ReadResponse] failed to "), proxy.Addr())
		return
		// do somethng nasty
	}
	debug(999, fmt.Sprintf("[Socks](%s)", ss.parent.Addr), color.YellowString("Connected to"), addr, "through", proxy.Addr(), "code:", resp.StatusCode, "from:", c.RemoteAddr().String())

	// a ok should tell the client that?
	// c.Write(Socks5StatusSucceeded)

	rh.Bin.cmd = 0x00 // success
	rh.WriteTo(c)

	debug(999, fmt.Sprintf("[Socks](%s)", ss.parent.Addr), color.GreenString("Start pipe to"), addr, "through", proxy.Addr(), "by:", c.RemoteAddr().String())

	// pipe content
	go transfer(destConn, c)
	transfer(c, destConn)

}
func ReadReqHeader(c net.Conn) (*SocksRequestHeader, error) {
	rh := SocksRequestHeader{}

	debug(9999, "[Socks]", "reading header")
	h1 := make([]byte, 4)
	n, err := c.Read(h1)
	if n != 4 || err != nil {
		return nil, err
	}
	cmd := h1[1]
	rh.Bin.ver = h1[0]
	rh.Bin.cmd = h1[1]
	debug(9999, "[Socks]", "cmd", int(cmd))
	// only connect allowed
	if cmd != 0x01 {
		return nil, errors.New("Only Connect")
	}
	atyp := h1[3]
	rh.Bin.atyp = h1[3]
	var addr string
	switch atyp {
	// if ipv4
	case 0x01:
		var b net.IP
		b = make([]byte, 4)
		n, err = c.Read(b)
		if n != 4 || err != nil {
			return nil, err
		}
		bn := []byte(b)
		rh.Bin.addr = &bn
		addr = b.String()
	// if domain
	case 0x03:
		addrl := make([]byte, 1)
		n, err = c.Read(addrl)
		if n != 1 || err != nil {
			return nil, err
		}
		addrb := make([]byte, int(addrl[0])+1)
		addrb[0] = addrl[0]
		n, err = c.Read(addrb[1:])
		if n != int(addrl[0]) || err != nil {
			return nil, err
		}
		rh.Bin.addr = &addrb
		addr = string(addrb[1:])

	// if ipv6
	case 0x04:
		var b net.IP
		b = make([]byte, 16)
		n, err = c.Read(b)
		if n != 16 || err != nil {
			return nil, err
		}
		bn := []byte(b)
		rh.Bin.addr = &bn
		addr = b.String()
	}
	debug(9999, "[Socks]", "addr", atyp, addr)
	portb := make([]byte, 2)
	rh.Bin.port = portb
	n, err = c.Read(portb)
	if n != 2 || err != nil {
		return nil, err
	}
	port := binary.BigEndian.Uint16(portb)
	rh.Addr = addr
	rh.Port = int(port)
	debug(9999, "[Socks]", "port", port)

	return &rh, nil
}
func SocksPassAuth(c net.Conn) (*netutils.UserInfo, error) {
	defer debug(9999, "[Socks]", "Read pass auth done")
	h1 := make([]byte, 2)
	n, err := c.Read(h1)
	if n != 2 || err != nil {
		return nil, err
	}

	uname := make([]byte, h1[1])
	n, err = c.Read(uname)

	if n != int(h1[1]) || err != nil {
		return nil, err
	}
	plen := make([]byte, 1)
	n, err = c.Read(plen)

	if n != 1 || err != nil {
		return nil, err
	}
	passwd := make([]byte, plen[0])
	n, err = c.Read(passwd)

	if n != int(plen[0]) || err != nil {
		return nil, err
	}
	return &netutils.UserInfo{
		User: string(uname),
		Pass: string(passwd),
	}, nil

}

func socksDom(dom string) []byte {
	d := []byte(dom)
	return append([]byte{uint8(len(d))}, d...)
}
