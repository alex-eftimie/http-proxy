package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/alex-eftimie/netutils"
	"github.com/alex-eftimie/utils"
	"github.com/davecgh/go-spew/spew"
	"github.com/fatih/color"
	"golang.org/x/net/http2"
)

// H2CServer is for the proxy accelerator, it support proxying on h2c connections
type H2CServer struct {
	*http2.Server
	parent *Server
}

func newH2CServer(s *Server) *H2CServer {
	h2s := &H2CServer{
		parent: s,
	}
	h2s.Server = &http2.Server{}

	return h2s
}

// Serve serves connection on an existing listener
func (h2c *H2CServer) Serve(list net.Listener) error {

	for {
		rw, err := list.Accept()

		if err != nil {
			return err
		}

		color.Red("New connection established")
		cc := &netutils.CounterConn{Conn: rw, Upstream: 0, Downstream: 0}
		if utils.DebugLevel >= 9999 {
			cp := &netutils.PrinterConn{Conn: rw}
			cc = &netutils.CounterConn{Conn: cp, Upstream: 0, Downstream: 0}
		}
		// spew.Dump(cc, h2c)
		h2c.ServeConn(cc, &http2.ServeConnOpts{Handler: h2c})
	}
}

// TODO: potential improvement, only check authentication and/or bandwidth per connection, not per request
func (h2c *H2CServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	r.URL.Host = r.Host

	utils.Debugf(999, "[H2C](%s) m: %s, h: %s, r: %s", h2c.parent.Addr, r.Method, r.Host, r.RemoteAddr)

	// h2c connections can't be hijacked, but we don't need to we can hack
	// the request and response bodies and achieve the same result

	if h2c.parent.limiter.Add() == false {
		http.Error(w, "Too many threads", http.StatusPaymentRequired)
		return
	}
	defer h2c.parent.limiter.Done()

	if h2c.parent.IsExpired() {
		http.Error(w, "Expired", http.StatusPaymentRequired)
		return
	}

	if h2c.parent.HasBW() == false {
		http.Error(w, "Not enough bandwidth", http.StatusPaymentRequired)
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

	if err := h2c.parent.CheckAuth(uinfo, ip); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	upstreamHost, upstreamPort := netutils.GetHostPort(r)
	tunnel, proxy, err := h2c.parent.GetProxyAndTunnel(uinfo, m, upstreamHost, upstreamPort)
	if err != nil || tunnel == nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	defer func() {
		if tunnel != nil {
			tunnel.Close()
		}
	}()
	defer h2c.parent.RunAccountant("H2C", nil, tunnel)

	if err != nil || tunnel == nil {
		http.Error(w, "Unreachable", http.StatusBadGateway)
		utils.Debugf(999, "[H2C](%s): host: %s, port: %d, proxy: [%s]%s, Err: Unreachable, %s", r.RemoteAddr, upstreamHost, upstreamPort, proxy.Type, proxy.Addr(), err)
		return
	}

	// if len(proxy.User) != 0 || len(proxy.Pass) != 0 {
	// 	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(proxy.Auth()))
	// 	r.Header.Set("Proxy-Authorization", basicAuth)
	// } else {
	r.Header.Del("Proxy-Authorization")
	// }

	// Write the Request to the Upstream Proxy
	if r.Method != "CONNECT" {
		// r.WriteProxy(os.Stderr)
		if r.URL.Scheme == "https" {
			counterConn := tunnel
			config := &tls.Config{InsecureSkipVerify: true}
			tunnel = tls.Client(tunnel, config)

			// this must run before the accountant so we can get the total bytes read
			defer func() {
				tunnel = counterConn
			}()
		}
		r.Write(tunnel)

		reader := bufio.NewReader(tunnel)
		resp, err := http.ReadResponse(reader, r)
		if err != nil {
			spew.Dump(resp)
			log.Println("Proxy Forwarding Error:"+err.Error(), http.StatusServiceUnavailable)
			http.Error(w, "Proxy Forwarding Error:"+err.Error(), http.StatusServiceUnavailable)
			return
		}
		if resp.Header.Get("Proxy-Agent") != "" {
			resp.Header.Del("Proxy-Agent")
			resp.Header.Set("X-Proxy-Agent", Co.ProxyAgent)
		}

		// resp.Write(os.Stderr)

		netutils.CopyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		writer := flushWriter{w}
		io.Copy(writer, resp.Body)
		resp.Body.Close()
		tunnel.Close()
		return
	}
	w.WriteHeader(200)

	writer := flushWriter{w}
	writer.Flush()

	data := netutils.HTTPReadWriteCloser{
		Writer:     writer,
		ReadCloser: r.Body,
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, netutils.ContextKeyPipeTimeout, time.Duration(Co.ReadWriteTimeout)*time.Second)
	netutils.RunPiper(ctx, data, tunnel)
	color.Yellow("Piper Finished")
}
func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

// bidirectionalReadWriter is
type bidirectionalReadWriter struct {
	http.ResponseWriter
	*BDH2C
}

// Close closes the connection
func (b bidirectionalReadWriter) Close() error {
	return b.BDH2C.Close()
}

// Close closes the connection
func (b bidirectionalReadWriter) Read(p []byte) (n int, e error) {
	return b.BDH2C.Read(p)
}

// Close closes the connection
func (b bidirectionalReadWriter) Write(p []byte) (n int, e error) {
	return b.BDH2C.Write(p)
}

func (b bidirectionalReadWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return b.BDH2C, nil, nil
}

// BDH2C bidirectional http2 clear text stream
type BDH2C struct {
	flushWriter
	io.ReadCloser
	*netutils.CounterConn
}

// Close closes the connection
func (b BDH2C) Close() error {
	return b.ReadCloser.Close()
}

// Close closes the connection
func (b BDH2C) Read(p []byte) (n int, e error) {
	return b.ReadCloser.Read(p)
}

// Close closes the connection
func (b BDH2C) Write(p []byte) (n int, e error) {
	return b.flushWriter.Write(p)
}

// GetCounterConn returns a pointer to the internal
func (b BDH2C) GetCounterConn() *netutils.CounterConn {
	return b.CounterConn
}

type flushWriter struct {
	w io.Writer
}

func (fw flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	fw.w.(http.Flusher).Flush()
	return n, err
}
func (fw flushWriter) Flush() {
	fw.w.(http.Flusher).Flush()
}
