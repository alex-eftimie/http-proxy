package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/Alex-Eftimie/netutils"
	"github.com/Alex-Eftimie/utils"
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
	resp.Header.Add("X-Error", msg)
	resp.Write(c.Conn)
	c.Conn.Close()
	utils.Debugf(99, "[HTTP](%s): %s", c.Request.RemoteAddr, msg)
}
func (cs *CustomHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	utils.Debugf(999, "[HTTP](%s) m: %s, h: %s", cs.parent.Addr, r.Method, r.Host)
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

	uinfo := GetAuth(r)
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
		conn.httpError(err.Error(), http.StatusUnauthorized)
		return
	}

	proxy, err := cs.parent.SelectProxy(uinfo, m)

	if proxy == nil || err != nil {
		conn.httpError("Proxy Offline", http.StatusBadGateway)
		return
	}

	upstreamHost, upstreamPort := netutils.GetHostPort(r)

	tunnel, err := proxy.GetTunnel(upstreamHost, upstreamPort)

	if err != nil || tunnel == nil {
		conn.httpError("Proxy Unreachable", http.StatusBadGateway)
		utils.Debugf(999, "[HTTP](%s): Proxy Destination: %s", r.RemoteAddr, proxy.Addr())
		return
	}

	defer cs.parent.RunAccountant("HTTP", clientConn.(*cmux.MuxConn).Conn, tunnel)

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

		cs.parent.RunPiper(clientConn, tunnel)

	} else { // GET, POST, OPTIONS, etc...
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		nr := r.Clone(ctx)

		if len(proxy.User) != 0 || len(proxy.Pass) != 0 {
			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(proxy.Auth()))
			nr.Header.Set("Proxy-Authorization", basicAuth)
		} else {
			nr.Header.Del("Proxy-Authorization")
		}

		reader := bufio.NewReader(tunnel)

		// Write the Request to the Upstream Proxy
		nr.WriteProxy(tunnel)

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
