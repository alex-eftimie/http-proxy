package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alex-eftimie/netutils"
	"github.com/alex-eftimie/socks5"
	"github.com/alex-eftimie/utils"
	"github.com/fatih/color"
	"github.com/pkg/errors"
)

const (
	// TypeHTTP is used for HTTP upstream proxies
	TypeHTTP proxyType = "HTTP"

	// TypeSocks5 is used for Socks5 upstream proxies
	TypeSocks5 proxyType = "Socks5"
)

type proxyType string

// ProxyInfo forwards represents a http proxy
type ProxyInfo struct {
	User       string
	Pass       string
	Host       string
	Port       int
	Type       proxyType
	connection net.Conn
}

// ProxyQueue Holds a queue of pre-connected proxies
type ProxyQueue chan *ProxyInfo

// Clone returns a pointer to a new instance of the proxyinfo
// it does not clone the connection
func (pi *ProxyInfo) Clone() *ProxyInfo {
	return &ProxyInfo{
		User: pi.User,
		Pass: pi.Pass,
		Host: pi.Host,
		Port: pi.Port,
		Type: pi.Type,
	}
}

// Connect initializes a new connection
func (pi *ProxyInfo) Connect() (net.Conn, error) {
	dc, err := net.DialTimeout("tcp", pi.Addr(), 10*time.Second)
	if err != nil {
		utils.Debugf(999, "Proxy Upstream net.DialTimeout error: %s", err)
		return nil, err
	}
	if Co.DebugLevel > 9999 {
		dc = netutils.PrinterConn{Conn: dc, Prefix: "UpstreamProxy"}
	}
	pi.connection = dc
	return dc, nil
}

// GetTunnel connects to the proxy and establishes a tunnel
func (pi *ProxyInfo) GetTunnel(host string, port int) (net.Conn, error) {
	if pi.Type == TypeSocks5 {
		var auth *netutils.UserInfo
		if len(pi.User) != 0 || len(pi.Pass) != 0 {
			auth = &netutils.UserInfo{
				User: pi.User,
				Pass: pi.Pass,
			}
		}

		cl := &socks5.Client{
			Auth: auth,
			Conn: pi.connection,
		}
		conn, err := cl.Open(pi.Addr())

		if err != nil {
			return nil, err
		}
		err = conn.Connect(host, port)
		if err != nil {
			conn.Close()
			return nil, err
		}
		return conn.Conn, nil
	}

	// HTTP Proxy
	// dc, err := net.DialTimeout("tcp", pi.Addr(), 10*time.Second)
	// if err != nil {
	// 	return nil, err
	// }

	// we already have the connection from the pre-connect
	dc := pi.connection

	// pc := &netutils.PrinterConn{Conn: dc, Prefix: "tunnel"}
	destConn := &netutils.CounterConn{Conn: dc, Upstream: 0, Downstream: 0}
	hostport := fmt.Sprintf("%s:%d", host, port)
	nr := &http.Request{
		Method:     "CONNECT",
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL: &url.URL{
			Host: hostport,
		},
		Host:   hostport,
		Header: make(http.Header),
	}

	if len(pi.User) != 0 || len(pi.Pass) != 0 {
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(pi.Auth()))
		nr.Header.Set("Proxy-Authorization", basicAuth)
	}

	reader := bufio.NewReader(destConn)

	// Write the Request to the Upstream Proxy
	nr.WriteProxy(destConn)

	resp, err := http.ReadResponse(reader, nr)
	if Co.DebugLevel >= 9999 {
		if resp != nil {
			// resp.Write(os.Stderr)
		} else {
			log.Printf("Could not reach upstream proxy, connect[%s:%d] error: %s", host, port, err)
		}
	}
	if err != nil || resp.StatusCode != http.StatusOK {
		if err == nil {
			err = errors.Errorf("%d %s", resp.StatusCode, resp.Status)
		}
		destConn.Close()
		color.Red("%s", err)
		return nil, err
	}

	return destConn, nil
}

// ReadProxy returns a pointer to a new instance of ProxyInfo parsed from proxy
// @param proxy string format user:pass@ip:port
func ReadProxy(proxy string) *ProxyInfo {

	r, _ := regexp.Compile("^(?P<type>(http|socks5)://){0,1}((?P<user>[^:@]+)(:(?P<pass>[^:@]+)){0,}@){0,1}(?P<host>[0-9A-Za-z\\.\\-]+):(?P<port>[0-9]+)")
	res := r.FindStringSubmatch(proxy)
	result := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = res[i]
		}
	}
	var tp proxyType
	if v, ok := result["type"]; !ok {
		tp = "HTTP"
	} else {
		if v == "socks5://" {
			tp = TypeSocks5
		} else {
			tp = TypeHTTP
		}
	}

	port, _ := strconv.Atoi(result["port"])
	return &ProxyInfo{
		User: result["user"],
		Pass: result["pass"],
		Host: result["host"],
		Port: port,
		Type: tp,
	}
}

// GetRandomProxy returns a random proxy from the Proxy Cache
func GetRandomProxy(group string) *ProxyInfo {

	// if the group does not exist, skip
	if _, ok := Ca.Proxies[group]; !ok {
		log.Println("Proxy group", group, "does not exist")
		return nil
	}

	go proxyRequest(group)
	// get a proxy from the queue

	return getProxyFromQueue(group)
}

func proxyRequest(group string) {
	var err error
	var proxy *ProxyInfo

	defer func() {
		if proxy != nil {
			Ca.ProxyQueueMap[group] <- proxy
			return
		}
		log.Println(color.RedString("Could not get a proxy, err:"), err)
		go proxyRequest(group)
	}()

	var randomIndex int
	// send a proxy request on the queue
	proxies := len(Ca.Proxies[group])
	if proxies == 0 {
		return
	}

	if proxies == 1 {
		randomIndex = 0
	} else {
		randomIndex = rand.Intn(proxies)
	}

	pi := Ca.Proxies[group][randomIndex]

	pn := pi.Clone()

	_, err = pn.Connect()
	if err == nil {
		proxy = pn
	} else {
		proxy = nil
	}
}
func getProxyFromQueue(group string) *ProxyInfo {
get:
	x := <-Ca.ProxyQueueMap[group]
	if x.connection == nil || connCheck(x.connection) != nil {
		utils.Debugf(9999, "Pooled proxy connection already closed")
		goto get
	}
	utils.Debugf(9999, color.GreenString("Pooled proxy connection seems alive"))
	return x
}

// ParseParams parses param for _Key?-Value pairs and returns a map
// @param param string is the initial string
// @param *map[string]string holds the key, value pairs
// @returns ret string the initial string with all the key, value pairs removed
func ParseParams(param string, mp *map[string]string) string {

	m := *mp
	spl := strings.Split(param, "_")
	if len(spl) < 2 {
		return ""
	}
	ret := spl[0]
	spl = spl[1:]
	for _, cf := range spl {
		ss := strings.SplitN(cf, "-", 2)
		if len(ss) < 2 {
			m[ss[0]] = "1"
		} else {
			ss[1] = strings.ToUpper(ss[1])
			m[ss[0]] = ss[1]
		}
	}
	return ret
}

// Addr returns a string Host:Port
func (pi *ProxyInfo) Addr() string {
	return fmt.Sprintf("%s:%d", pi.Host, pi.Port)
}

// Auth returns a string User:Pass
func (pi *ProxyInfo) Auth() string {
	return fmt.Sprintf("%s:%s", pi.User, pi.Pass)
}

func proxyQueueManager(group string) {
	for {
		for len(Ca.ProxyQueueMap[group]) < Co.MinPreConnections {
			utils.Debug(999, "Preloading proxy", group)
			proxyRequest(group)
		}
		time.Sleep(3 * time.Millisecond)
	}
}
func connCheck(conn net.Conn) error {
	var sysErr error = nil
	if _, ok := conn.(netutils.PrinterConn); ok {
		conn = conn.(netutils.PrinterConn).Conn
	}
	rc, err := conn.(syscall.Conn).SyscallConn()
	if err != nil {
		return err
	}
	err = rc.Read(func(fd uintptr) bool {
		var buf []byte = []byte{0}
		n, _, err := syscall.Recvfrom(int(fd), buf, syscall.MSG_PEEK|syscall.MSG_DONTWAIT)
		switch {
		case n == 0 && err == nil:
			sysErr = io.EOF
		case err == syscall.EAGAIN || err == syscall.EWOULDBLOCK:
			sysErr = nil
		default:
			sysErr = err
		}
		return true
	})
	if err != nil {
		return err
	}

	return sysErr
}
