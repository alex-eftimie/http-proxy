package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/shenwei356/util/bytesize"
)

// PrettyByte is used to store Bytes in Value bytes and Readable format
type PrettyByte struct {
	Value    int64
	Readable string
}

var (
	// AuthTypeIP Auth.Type is AuthTypeIP when ip authentication is used
	AuthTypeIP = "IP"

	// AuthTypeUserPass Auth.Type is AuthTypeUserPass when user+pass authentication is used
	AuthTypeUserPass = "UserPass"
)
var SessionMaster *SessionManager

func init() {
	SessionMaster = &SessionManager{
		Sessions: make(map[string]*hp),
	}
}

// Auth holds the Server Auth data
type Auth struct {
	AuthToken string
	Type      string
	User      string
	Pass      string
	IP        map[string]bool
}

// Server is a single Proxy
type Server struct {
	ID         string
	Server     *http.Server `json:"-"`
	Client     string
	Addr       string
	Auth       Auth
	Bytes      *PrettyByte
	MaxThreads *int
	mux        sync.Mutex
	parent     *ConfigType // don't export it, it will cause cycles
	limiter    *ServerLimiter
}

// RunServer starts a http listener on s.Addr
func RunServer(s *Server) error {
	Ca.Lock()
	defer Ca.Unlock()
	if s.Auth.Type != AuthTypeIP && s.Auth.Type != AuthTypeUserPass {
		log.Fatalf("Server %s has invalid Auth.Type %s", s.ID, s.Auth.Type)
	}

	s.parent = Co
	if s.ID != "" {
		Ca.ServerMap[s.ID] = s
	}

	if s.Auth.AuthToken != "" {
		Ca.ServerMap[s.Auth.AuthToken] = s
	}
	Ca.ServerMap[s.Addr] = s

	s.SyncBandwidth()

	hs := &http.Server{
		Addr:    s.Addr,
		Handler: s,
	}

	s.limiter = &ServerLimiter{max: s.MaxThreads}

	s.Server = hs

	return ListenAndServe(hs)

}

// ListenAndServe listens on srv.Addr and starts a goroutine to Serve connections
func ListenAndServe(srv *http.Server) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("Failed to listen", err)
		return err
	}
	cl := CounterListener{Listener: ln}
	go func() {
		if r := recover(); r != nil {
			log.Println("Recovered in f", r)
		}
		srv.Serve(cl)
	}()

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if s.limiter.Add() == false {
		w.Header().Set("X-Error", "Too many threads")
		debug(999, "Too many threads:", r.RemoteAddr)
		http.Error(w, "", http.StatusPaymentRequired)
		return
	}
	defer s.limiter.Done()

	debug(999, "New connection:", r.RemoteAddr)
	if s.HasBW() == false {
		w.Header().Set("X-Error", "Not enough bandwidth")
		debug(999, "Not enough bandwidth:", r.RemoteAddr)
		http.Error(w, "", http.StatusPaymentRequired)
		return
	}
	w.Header().Set("X-Proxy-Agent", Co.ProxyAgent)

	uinfo := GetAuth(r)
	m := make(map[string]string)

	if uinfo == nil {
		uinfo = &UserInfo{User: "", Pass: ""}
	} else {
		if uinfo.User != "" {
			r := ParseParams(uinfo.User, &m)
			if r != "" {
				uinfo.User = r
			}
		}

		if uinfo.Pass != "" {
			r := ParseParams(uinfo.Pass, &m)
			if r != "" {
				uinfo.Pass = r
			}
		}
	}

	ProxyConfig := r.Header.Get("Proxy-Config")
	if ProxyConfig != "" {
		r.Header.Del("Proxy-Config")
		ParseParams(ProxyConfig, &m)
	}

	if s.Auth.Type == "IP" {
		ip := strings.Split(r.RemoteAddr, ":")[0]

		if val, ok := s.Auth.IP[ip]; !ok || val == false {
			w.Header().Set("X-Error", "IP not allowed")
			w.Header().Add("X-ip", ip)
			http.Error(w, "", http.StatusProxyAuthRequired)
			debug(999, "IP not allowed:", r.RemoteAddr)
			return
		}
	} else if s.Auth.Type == "UserPass" {
		if CheckUser(uinfo, s.Auth) == false {

			w.Header().Set("Proxy-Authenticate", " Basic")

			w.WriteHeader(http.StatusProxyAuthRequired)

			debug(999, r.RemoteAddr, spew.Sdump(r.Header))
			debug(99, "Wrong User/Pass Combination:",
				"\n\tActual:", uinfo.User, uinfo.Pass,
				"\n\tExpected", s.Auth.Type, s.Auth.User, s.Auth.Pass)
			return
		}
	}
	var proxyP *HTTPProxy

	// IF sticky ip requested, get the same proxy
	if v, ok := m["sticky"]; ok {
		key := uinfo.User + "::" + v

		log.Println("get", key)
		proxyP = SessionMaster.GetSession(key)
		if proxyP == nil {
			log.Println("set", key)
			proxyP = GetRandomProxy()
			SessionMaster.SetSession(key, proxyP)
		}
	} else {
		log.Println("random proxy")
		proxyP = GetRandomProxy()
	}
	log.Println(proxyP.Port)
	return

	m["User"] = proxyP.User
	m["Pass"] = proxyP.Pass

	ParseProxyParams(&m)

	proxy := &HTTPProxy{
		Host: proxyP.Host,
		Port: proxyP.Port,
		User: m["User"],
		Pass: m["Pass"],
	}

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

	dc, err := net.DialTimeout("tcp", proxy.Addr(), 10*time.Second)
	if err != nil {

		resp := &http.Response{
			StatusCode: http.StatusBadGateway,
			Status:     http.StatusText(http.StatusBadGateway),
			Close:      true,
			ProtoMajor: r.ProtoMajor,
			ProtoMinor: r.ProtoMinor,

			Body: ioutil.NopCloser(bytes.NewBufferString("")),
		}

		resp.Write(clientConn)
		clientConn.Close()
		debug(999, "Proxy Dial Error:", err.Error(), r.RemoteAddr)
		return
	}
	destConn := &CounterConn{dc, 0, 0}

	// Count all bytes transfered no matter what
	defer func() {
		cl := clientConn.(*CounterConn)

		up := destConn
		// log.Println("Done", s.Addr, dd.Downstream, dd.Upstream)

		if up.Upstream+up.Downstream > cl.Upstream+cl.Downstream {
			log.Println("Larger upstream bandwidth")
			s.Consume(up.Downstream, up.Upstream)
		} else {
			s.Consume(cl.Downstream, cl.Upstream)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	nr := r.Clone(ctx)

	if len(proxy.User) != 0 || len(proxy.Pass) != 0 {
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(proxy.Auth()))
		nr.Header.Set("Proxy-Authorization", basicAuth)
	} else {
		nr.Header.Del("Proxy-Authorization")
	}

	reader := bufio.NewReader(destConn)

	// Write the Request to the Upstream Proxy
	nr.WriteProxy(destConn)

	resp, err := http.ReadResponse(reader, nr)

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		debug(99, "Upstream read Error:", r.RemoteAddr, err.Error())
		if resp != nil && resp.Header != nil {
			debug(999, spew.Sdump(resp.Header))
		}
		return
	}

	if resp.Header.Get("Proxy-Agent") != "" {
		resp.Header.Del("Proxy-Agent")
		resp.Header.Set("X-Proxy-Agent", Co.ProxyAgent)
	}

	if r.Method == http.MethodConnect {
		resp.Header.Set("X-Proxy-Agent", Co.ProxyAgent)

		// if proxy error, bail with a generic bad gateway
		if resp.StatusCode != 200 {
			resp.StatusCode = http.StatusBadGateway
			resp.Status = http.StatusText(http.StatusBadGateway)
			resp.Write(clientConn)
			clientConn.Close()
			return
		}

		// if we don't close the body, write will just hang
		resp.Body.Close()
		resp.Write(clientConn)

		// pipe content
		go transfer(destConn, clientConn)
		transfer(clientConn, destConn)

	} else {
		resp.Write(clientConn)

		// if we don't have a content-length set, it won't automatically close
		if resp.ContentLength == -1 {
			clientConn.Close()
		}
	}
}

// SyncBandwidth sync Bytes.Value to Bytes.Readable if they are different
// Used when Updating bandwidth to update bandwidth in human readable format
func (s *Server) SyncBandwidth() {
	// Skip bandwidth for this port
	if s.Bytes != nil {

		val := bytesize.ByteSize(s.Bytes.Value)
		if s.Bytes.Readable != val.String() {
			newSize, err := bytesize.Parse([]byte(s.Bytes.Readable))
			if err != nil {
				log.Printf("Server %s error changing bandwidth: %s\n", s.Addr, err)
				return
			}
			log.Printf("Server %s changed size from %s to %s\n", s.Addr, val.String(), newSize.String())
			s.Bytes.Value = int64(newSize)
		}
	}
}

// HasBW checks if the Server still has bandwidth remaining
// returns true if bandwidth counting is disabled
func (s *Server) HasBW() bool {

	// No need to also lock parent, only reading
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.Bytes == nil {
		return true
	}

	if s.Bytes.Value > 0 {
		return true
	}

	return false
}

// Consume updates the server bandwith reducing Download bytes and Upload Bytes
func (s *Server) Consume(Download, Upload int64) *PrettyByte {

	// Lock the parent to prevent race update + save
	s.parent.Lock()
	s.mux.Lock()
	defer s.mux.Unlock()
	defer s.parent.Unlock()

	if s.Bytes == nil {
		return nil
	}

	s.Bytes.Value -= (Download + Upload)
	fmt.Print(s.Addr, ", Diff: ", fmt.Sprintf("%-10s", bytesize.ByteSize(Download+Upload)), ", Old Val: ", fmt.Sprintf("%-10s", s.Bytes.Readable))

	s.Bytes.Readable = bytesize.ByteSize(s.Bytes.Value).String()
	fmt.Print(", New Val: ", fmt.Sprintf("%-10s", s.Bytes.Readable), "\n")
	return s.Bytes
}

// Replenish replenishes bandwith for a server, if absoluteVal is true then bytes is reset to amount
// else bytes is increased by amount
// if amount is -1 && absoluteVal then bandwidth counting is disabled
func (s *Server) Replenish(amount int64, absoluteVal bool) {

	// Lock the parent to prevent race update + save
	s.parent.Lock()
	s.mux.Lock()
	defer s.mux.Unlock()
	defer s.parent.Unlock()

	if s.Bytes == nil && !absoluteVal {
		return
	}
	if amount == -1 && absoluteVal {
		s.Bytes = nil
		return
	}
	if absoluteVal {
		s.Bytes.Value = amount
	} else {
		s.Bytes.Value += amount
	}
	s.Bytes.Readable = bytesize.ByteSize(s.Bytes.Value).String()
}

// CloseServers as the name Implies
func CloseServers() {
	for _, server := range Ca.Servers {
		server.Server.Close()
		log.Println("Closed server", server.Addr)
	}
}

// transfer just copies from source to destination then closes both
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
