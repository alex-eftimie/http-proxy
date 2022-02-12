package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alex-eftimie/netutils"
	"github.com/alex-eftimie/utils"
	"github.com/soheilhy/cmux"
)

type CustomHTTPReq struct {
	net.Conn
	*http.Request
}
type CustomHTTPServer struct {
	*http.Server
	parent *Server
}

func newCustomHTTPServer(s *Server) *CustomHTTPServer {
	cs := &CustomHTTPServer{
		parent: s,
	}
	cs.Server = &http.Server{
		Addr:    s.Addr,
		Handler: cs,
	}

	return cs
}
func (c *CustomHTTPReq) httpError(msg string, status int) {
	resp := http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Close:      true,
		ProtoMajor: c.Request.ProtoMajor,
		ProtoMinor: c.Request.ProtoMinor,
		Body:       ioutil.NopCloser(bytes.NewBufferString("")),
		Header:     make(http.Header),
	}

	if status == http.StatusProxyAuthRequired {
		resp.Header.Add("Proxy-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", Co.ProxyAgent))
	}

	resp.Header.Add("X-Error", msg)
	resp.Write(c.Conn)
	c.Conn.Close()
	utils.Debugf(99, "[HTTP](%s): %s", c.Request.RemoteAddr, msg)
}
func (cs *CustomHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	utils.Debugf(999, "[HTTP](%s) m: %s, h: %s, r: %s", cs.parent.Addr, r.Method, r.Host, r.RemoteAddr)
	// hijack the HTTP Connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Tunneling(HJ) not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	conn := &CustomHTTPReq{
		Conn:    clientConn,
		Request: r,
	}

	if cs.parent.limiter.Add() == false {
		conn.httpError("Too many threads", http.StatusPaymentRequired)
		return
	}
	defer cs.parent.limiter.Done()

	if cs.parent.IsExpired() {
		conn.httpError("Expired", http.StatusPaymentRequired)
		return
	}

	if cs.parent.HasBW() == false {
		conn.httpError("Not enough bandwidth", http.StatusPaymentRequired)
		return
	}
	w.Header().Set("X-Proxy-Agent", Co.ProxyAgent)

	uinfo := netutils.GetAuth(r)
	m := make(map[string]string)

	if uinfo == nil {
		uinfo = &netutils.UserInfo{User: "", Pass: ""}
	} else {
		if uinfo.User != "" {
			r := utils.ParseParams(uinfo.User, &m, false)
			if r != "" {
				uinfo.User = r
			}
		}

		if uinfo.Pass != "" {
			r := utils.ParseParams(uinfo.Pass, &m, false)
			if r != "" {
				uinfo.Pass = r
			}
		}
	}

	ProxyConfig := r.Header.Get("Proxy-Config")
	if ProxyConfig != "" {
		r.Header.Del("Proxy-Config")
		utils.ParseParams(ProxyConfig, &m, false)
	}

	ip := strings.Split(r.RemoteAddr, ":")[0]

	if err := cs.parent.CheckAuth(uinfo, ip); err != nil {
		conn.httpError(err.Error(), http.StatusProxyAuthRequired)
		return
	}
	upstreamHost, upstreamPort := netutils.GetHostPort(r)
	// spew.Dump(upstreamHost, upstreamPort)
	// color.Green("%s:%d", upstreamHost, upstreamPort)
	// r.WriteProxy(os.Stderr)
	tunnel, _, err := cs.parent.GetProxyAndTunnel(uinfo, m, upstreamHost, upstreamPort)
	if err != nil {
		conn.httpError("Could not establish tunnel", http.StatusBadGateway)
		utils.Debugf(999, "[HTTP](%s) e: %s, m: %s, h: %s, r: %s", cs.parent.Addr, err.Error(), r.Method, r.Host, r.RemoteAddr)
		return
	}
	if tunnel == nil {
		conn.httpError("Proxy Unreachable", http.StatusBadGateway)
		return
	}

	var dc *netutils.CounterConn
	ac, isAccountable := clientConn.(Accountable)
	if !isAccountable {
		if mc, ok := clientConn.(*cmux.MuxConn); ok {
			dc = mc.Conn.(*netutils.CounterConn)
		} else {
			log.Fatalln("Not Accountable connection")
		}
	} else {
		dc = ac.GetCounterConn()
	}

	if Co.DebugLevel > 99 {
		defer func() {
			// if the accountant did not run, report it
			uc := tunnel.(*netutils.CounterConn)

			if uc.Downstream != -1 || dc.Downstream != -1 {
				reportError(fmt.Sprintf("Accountant did not run on connection : %d : %d", uc.Downstream, dc.Downstream))
			}
		}()
	}

	defer cs.parent.RunAccountant("HTTP", dc, tunnel)

	if r.Method == "CONNECT" {
		resp := http.Response{
			StatusCode: http.StatusOK,
			Status:     http.StatusText(http.StatusOK),
			Close:      true,
			ProtoMajor: r.ProtoMajor,
			ProtoMinor: r.ProtoMinor,
			Body:       ioutil.NopCloser(bytes.NewBufferString("")),
			Header:     make(http.Header),
		}
		resp.Header.Set("X-Proxy-Agent", Co.ProxyAgent)

		// if we don't close the body, write will just hang
		resp.Body.Close()
		resp.Write(clientConn)

		// cs.parent.RunPiper(clientConn, tunnel)
		ctx := context.Background()
		ctx = context.WithValue(ctx, netutils.ContextKeyPipeTimeout, time.Duration(Co.ReadWriteTimeout)*time.Second)
		netutils.RunPiper(ctx, clientConn, tunnel)

	} else { // GET, POST, OPTIONS, etc...
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		nr := r.Clone(ctx)

		// if len(proxy.User) != 0 || len(proxy.Pass) != 0 {
		// 	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(proxy.Auth()))
		// 	nr.Header.Set("Proxy-Authorization", basicAuth)
		// } else {
		nr.Header.Del("Proxy-Authorization")
		// }

		if nr.URL.Scheme == "https" {
			counterConn := tunnel
			config := &tls.Config{InsecureSkipVerify: true}
			tunnel = tls.Client(tunnel, config)

			// this must run before the accountant so we can get the total bytes read
			defer func() {
				tunnel = counterConn
			}()
		}
		reader := bufio.NewReader(tunnel)

		// Write the Request to the Upstream Server, we're already connected using CONNECT protocol
		// and if https is required, we've already set that up
		nr.Write(tunnel)

		resp, err := http.ReadResponse(reader, nr)
		if err != nil {
			conn.httpError("Proxy Forwarding Error:"+err.Error(), http.StatusServiceUnavailable)
			return
		}
		if resp.Header.Get("Proxy-Agent") != "" {
			resp.Header.Del("Proxy-Agent")
			resp.Header.Set("X-Proxy-Agent", Co.ProxyAgent)
		}
		resp.Write(clientConn)

		// if we don't have a content-length set, it won't automatically close
		if resp.ContentLength == -1 {
			clientConn.Close()
		}
	}

}
