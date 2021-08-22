package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Alex-Eftimie/netutils"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/shenwei356/util/bytesize"
	"github.com/soheilhy/cmux"
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

var AccessLogger *log.Logger

func init() {

	SessionMaster = &SessionManager{
		Sessions: make(map[string]*hp),
	}
	if Co.LogRequestsLevel >= 2 {
		// // If the file doesn't exist, create it, or append to the file
		// f, err := os.OpenFile("logs/access.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		// if err != nil {
		// 	log.Fatalln("Failed to open access log", err)
		// }
		logf, err := rotatelogs.New(
			"logs/access.log.%Y-%m-%d-%M",
			rotatelogs.WithLinkName("logs/access.log"),
			rotatelogs.WithRotationTime(24*time.Hour),
		)
		if err != nil {
			log.Fatalln("Failed to open log:", err)
		}

		w := io.MultiWriter(os.Stdout, logf)
		AccessLogger = log.New(w, "", log.LstdFlags|log.LUTC)
	} else if Co.LogRequestsLevel == 1 {
		AccessLogger = log.New(os.Stdout, "", log.LstdFlags|log.LUTC)
	} else {
		AccessLogger = log.New(ioutil.Discard, "", log.LstdFlags|log.LUTC)
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

type BWUsage struct {
	Upload   int64 `json:"U"`
	Download int64 `json:"D"`
}

// Server is a single Proxy
type Server struct {
	sync.Mutex
	ID string
	// HTTPServer *http.Server `json:"-"`
	// Socks5Server   *socks5Server `json:"-"`
	HTTPServer     *CustomHTTPServer   `json:"-"`
	Socks5Server   *CustomSocks5Server `json:"-"`
	CMux           cmux.CMux           `json:"-"`
	Client         string
	Addr           string
	Auth           Auth
	Bytes          *PrettyByte
	MaxThreads     *int
	MaxIPs         *int
	ExpireAt       *PTime
	parent         *ConfigType // don't export it, it will cause cycles
	limiter        *ServerLimiter
	Devices        map[string]bool `json:",omitempty"`
	BWUsageHistory map[time.Time]*BWUsage
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

	if len(s.BWUsageHistory) == 0 {
		s.BWUsageHistory = make(map[time.Time]*BWUsage)
	}

	// test port first
	// Create the main listener.
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}

	// hs := &http.Server{
	// 	Addr:    s.Addr,
	// 	Handler: s,
	// }

	s.limiter = &ServerLimiter{max: s.MaxThreads}

	s.HTTPServer = newCustomHTTPServer(s)
	// s.HTTPServer = hs
	// s.Socks5Server = &socks5Server{AuthType: s.Auth.Type, parent: s}
	s.Socks5Server = newSocks5Server(s)

	// return ListenAndServe(hs)
	go s.Serve(l)
	return nil
}

func (s *Server) Close() {
	// s.CMux.Close()
	s.HTTPServer.Close()
	s.Socks5Server.Close()
}
func (s *Server) Serve(l net.Listener) {

	cl := netutils.CounterListener{Listener: l}

	log.Println("[HTTP] Listening on", s.Addr)

	s.CMux = cmux.New(cl)

	// run the matchers
	socks5Matcher := s.CMux.Match(Socks5Matcher())
	httpMatcher := s.CMux.Match(cmux.HTTP1Fast())

	// run the servers
	go s.HTTPServer.Serve(httpMatcher)
	go s.Socks5Server.Serve(socks5Matcher)

	s.CMux.Serve()

	// // recover if weird errors pop up
	// go func() {
	// 	if r := recover(); r != nil {
	// 		log.Println("Recovered in f", r)
	// 	}
	// 	// srv.Serve(cl)
	// 	s.CMux.Serve()
	// }()
}

// // ListenAndServe listens on srv.Addr and starts a goroutine to Serve connections
// func ListenAndServe(srv *http.Server) error {
// 	addr := srv.Addr
// 	if addr == "" {
// 		addr = ":http"
// 	}
// 	ln, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		log.Println("Failed to listen", err)
// 		return err
// 	}
// 	cl := netutils.CounterListener{Listener: ln}
// 	go func() {
// 		if r := recover(); r != nil {
// 			log.Println("Recovered in f", r)
// 		}
// 		srv.Serve(cl)
// 	}()

// 	return nil
// }

// func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {

// 	if s.limiter.Add() == false {
// 		w.Header().Set("X-Error", "Too many threads")
// 		debug(999, "Too many threads:", r.RemoteAddr)
// 		http.Error(w, "", http.StatusPaymentRequired)
// 		return
// 	}
// 	defer s.limiter.Done()

// 	debug(999, "New connection:", r.RemoteAddr)

// 	if s.ExpireAt != nil && s.ExpireAt.Before(time.Now()) {
// 		w.Header().Set("X-Error", "Expired")
// 		debug(999, "Expired:", r.RemoteAddr)
// 		http.Error(w, "", http.StatusPaymentRequired)
// 		return
// 	}

// 	if s.HasBW() == false {
// 		w.Header().Set("X-Error", "Not enough bandwidth")
// 		debug(999, "Not enough bandwidth:", r.RemoteAddr)
// 		http.Error(w, "", http.StatusPaymentRequired)
// 		return
// 	}
// 	w.Header().Set("X-Proxy-Agent", Co.ProxyAgent)

// 	uinfo := GetAuth(r)
// 	m := make(map[string]string)

// 	if uinfo == nil {
// 		uinfo = &netutils.UserInfo{User: "", Pass: ""}
// 	} else {
// 		if uinfo.User != "" {
// 			r := utils.ParseParams(uinfo.User, &m, false)
// 			if r != "" {
// 				uinfo.User = r
// 			}
// 		}

// 		if uinfo.Pass != "" {
// 			r := utils.ParseParams(uinfo.Pass, &m, false)
// 			if r != "" {
// 				uinfo.Pass = r
// 			}
// 		}
// 	}

// 	ProxyConfig := r.Header.Get("Proxy-Config")
// 	if ProxyConfig != "" {
// 		r.Header.Del("Proxy-Config")
// 		utils.ParseParams(ProxyConfig, &m, false)
// 	}

// 	if s.Auth.Type == AuthTypeIP {
// 		ip := strings.Split(r.RemoteAddr, ":")[0]

// 		if val, ok := s.Auth.IP[ip]; !ok || val == false {
// 			w.Header().Set("X-Error", "IP not allowed")
// 			w.Header().Add("X-ip", ip)
// 			http.Error(w, "", http.StatusProxyAuthRequired)
// 			debug(999, "IP not allowed:", r.RemoteAddr)
// 			return
// 		}
// 	} else if s.Auth.Type == AuthTypeUserPass {
// 		if CheckUser(uinfo, s.Auth) == false {

// 			w.Header().Set("Proxy-Authenticate", " Basic")

// 			w.WriteHeader(http.StatusProxyAuthRequired)

// 			debug(999, r.RemoteAddr, spew.Sdump(r.Header))
// 			debug(99, "Wrong User/Pass Combination:",
// 				"\n\tActual:", uinfo.User, uinfo.Pass,
// 				"\n\tExpected", s.Auth.Type, s.Auth.User, s.Auth.Pass)
// 			return
// 		}
// 	}
// 	debug(999, "Auth Success", s.Auth.Type, r.RemoteAddr, "Server:", s.Auth.Pass, s.Auth.User, "User:", uinfo.Pass, uinfo.User)

// 	var proxyP *ProxyInfo

// 	m["Group"] = "Default"

// 	ParseProxyParams(&m, true)

// 	m["Group"] = strings.ToUpper(m["Group"])

// 	if s.Devices != nil {
// 		// if requested specific device
// 		d, ok := m["device"]
// 		if !ok {
// 			for v := range s.Devices {
// 				d = v
// 				break
// 			}
// 			m["device"] = d
// 		}
// 		proxyP = getDynamicProxy(d)
// 	} else if v, ok := m["sticky"]; ok { // IF sticky ip requested, get the same proxy
// 		key := uinfo.User + "::" + v

// 		// log.Println("get", key)
// 		proxyP = SessionMaster.GetSession(key)
// 		if proxyP == nil {
// 			// log.Println("set", key)
// 			proxyP = GetRandomProxy(m["Group"])
// 			SessionMaster.SetSession(key, proxyP)
// 		}
// 	} else {
// 		// log.Println("random proxy")
// 		proxyP = GetRandomProxy(m["Group"])
// 	}

// 	if proxyP == nil {
// 		http.Error(w, "Proxy Not Available", http.StatusBadGateway)
// 		return
// 	}

// 	m["User"] = proxyP.User
// 	m["Pass"] = proxyP.Pass
// 	ParseProxyParams(&m, false)

// 	proxy := &ProxyInfo{
// 		Host: proxyP.Host,
// 		Port: proxyP.Port,
// 		User: m["User"],
// 		Pass: m["Pass"],
// 		Type: proxyP.Type,
// 	}

// 	// hijack the HTTP Connection
// 	hijacker, ok := w.(http.Hijacker)
// 	if !ok {
// 		http.Error(w, "Tunneling(HJ) not supported", http.StatusInternalServerError)
// 		return
// 	}
// 	clientConn, _, err := hijacker.Hijack()
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusServiceUnavailable)
// 		return
// 	}
// 	defer clientConn.Close()

// 	var destConn *netutils.CounterConn
// 	done := make(chan bool, 1)

// 	if proxy.Type == TypeSocks5 {
// 		destConn = forwardSocks(proxy, r, clientConn, done)
// 	} else {
// 		destConn = forwardHTTP(proxy, r, clientConn, done)
// 	}

// 	// Count all bytes transfered no matter what
// 	// perform connection logging
// 	defer func() {
// 		// wait for the forwarding to finish
// 		<-done
// 		mc := clientConn.(*cmux.MuxConn)
// 		cl := mc.Conn.(*netutils.CounterConn)

// 		up := destConn

// 		if up.Upstream+up.Downstream > cl.Upstream+cl.Downstream {
// 			s.LogConnection(r.Host, "HTTP", up.Downstream, up.Upstream)
// 			s.Consume(up.Downstream, up.Upstream)
// 		} else {
// 			s.LogConnection(r.Host, "HTTP", cl.Downstream, cl.Upstream)
// 			s.Consume(cl.Downstream, cl.Upstream)
// 		}

// 	}()
// }

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
	s.Lock()
	defer s.Unlock()

	if s.Bytes == nil {
		return true
	}

	if s.Bytes.Value > 0 {
		return true
	}

	return false
}

// LogConnection logs the connection information
// Possible sort keys:
//		Connection Count,
//		Total upstream bytes
//		Total downstream bytes
//		Total Bytes
//
// One way to achieve this is to only log to file straight up connections
// and to have another code parse through the log file
// the beauty of this MO is that you can pass through the raw info as many times
// as needed and you can model the data however you want, including with other
// software
func (s *Server) LogConnection(host, payload string, down, up int64) {
	AccessLogger.Println(payload, s.ID, s.Addr, host, "⬆", up, "⬇", down)
}

// Consume updates the server bandwith reducing Download bytes and Upload Bytes
func (s *Server) Consume(Download, Upload int64) *PrettyByte {

	// Lock the parent to prevent race update + save
	s.parent.Lock()
	s.Lock()
	defer s.Unlock()
	defer s.parent.Unlock()

	if s.Bytes == nil {
		return nil
	}

	st := Bod(nil)

	usage, ok := s.BWUsageHistory[st]
	if !ok {
		usage = &BWUsage{}
		s.BWUsageHistory[st] = usage
	}
	usage.Upload += Upload
	usage.Download += Download

	s.Bytes.Value -= (Download + Upload)
	// fmt.Print(s.Addr, ", Diff: ", fmt.Sprintf("%-10s", bytesize.ByteSize(Download+Upload)), ", Old Val: ", fmt.Sprintf("%-10s", s.Bytes.Readable))

	s.Bytes.Readable = bytesize.ByteSize(s.Bytes.Value).String()
	// fmt.Print(", New Val: ", fmt.Sprintf("%-10s", s.Bytes.Readable), "\n")
	return s.Bytes
}

// Replenish replenishes bandwith for a server, if absoluteVal is true then bytes is reset to amount
// else bytes is increased by amount
// if amount is -1 && absoluteVal then bandwidth counting is disabled
func (s *Server) Replenish(amount int64, absoluteVal bool) {

	// Lock the parent to prevent race update + save
	s.parent.Lock()
	s.Lock()
	defer s.Unlock()
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
		server.Close()
		log.Println("Closed server", server.Addr)
	}
}

func Bod(t *time.Time) time.Time {
	if t == nil {
		n := time.Now()
		t = &n
	}
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

// func (s *Server) Lock() {
// 	log.Println("Locking Server", s.ID)
// 	s.Lock()
// }
// func (s *Server) Unlock() {
// 	log.Println("Unlocking Server", s.ID)
// 	s.Unlock()
// }

// func forwardHTTP(proxy *ProxyInfo, r *http.Request, clientConn net.Conn, done chan bool) *netutils.CounterConn {
// 	dc, err := net.DialTimeout("tcp", proxy.Addr(), 10*time.Second)
// 	if err != nil {

// 		resp := &http.Response{
// 			StatusCode: http.StatusBadGateway,
// 			Status:     http.StatusText(http.StatusBadGateway),
// 			Close:      true,
// 			ProtoMajor: r.ProtoMajor,
// 			ProtoMinor: r.ProtoMinor,
// 			Body:       ioutil.NopCloser(bytes.NewBufferString("")),
// 		}

// 		resp.Write(clientConn)
// 		clientConn.Close()
// 		debug(999, "Proxy Dial Error:", err.Error(), r.RemoteAddr, proxy.Addr())
// 		done <- true
// 		return nil
// 	}
// 	destConn := &netutils.CounterConn{dc, 0, 0}

// 	go func() {
// 		defer func() { done <- true }()
// 		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
// 		defer cancel()
// 		nr := r.Clone(ctx)

// 		if len(proxy.User) != 0 || len(proxy.Pass) != 0 {
// 			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(proxy.Auth()))
// 			nr.Header.Set("Proxy-Authorization", basicAuth)
// 		} else {
// 			nr.Header.Del("Proxy-Authorization")
// 		}

// 		reader := bufio.NewReader(destConn)

// 		// Write the Request to the Upstream Proxy
// 		nr.WriteProxy(destConn)

// 		resp, err := http.ReadResponse(reader, nr)

// 		if err != nil {
// 			resp := &http.Response{
// 				StatusCode: http.StatusServiceUnavailable,
// 				Status:     http.StatusText(http.StatusServiceUnavailable),
// 				Close:      true,
// 				ProtoMajor: r.ProtoMajor,
// 				ProtoMinor: r.ProtoMinor,
// 				Body:       ioutil.NopCloser(bytes.NewBufferString("")),
// 			}

// 			resp.Write(clientConn)

// 			debug(99, "Upstream read Error:", r.RemoteAddr, err.Error())
// 			if resp != nil && resp.Header != nil {
// 				debug(999, spew.Sdump(resp.Header))
// 			}
// 			return
// 		}

// 		if resp.Header.Get("Proxy-Agent") != "" {
// 			resp.Header.Del("Proxy-Agent")
// 			resp.Header.Set("X-Proxy-Agent", Co.ProxyAgent)
// 		}

// 		if r.Method == http.MethodConnect {
// 			resp.Header.Set("X-Proxy-Agent", Co.ProxyAgent)

// 			// if proxy error, bail with a generic bad gateway
// 			if resp.StatusCode != 200 {
// 				resp.StatusCode = http.StatusBadGateway
// 				resp.Status = http.StatusText(http.StatusBadGateway)
// 				resp.Write(clientConn)
// 				clientConn.Close()
// 				return
// 			}

// 			// if we don't close the body, write will just hang
// 			resp.Body.Close()
// 			resp.Write(clientConn)

// 			// pipe content
// 			go transfer(destConn, clientConn)
// 			transfer(clientConn, destConn)

// 		} else {
// 			resp.Write(clientConn)

// 			// if we don't have a content-length set, it won't automatically close
// 			if resp.ContentLength == -1 {
// 				clientConn.Close()
// 			}
// 		}
// 	}()
// 	return destConn
// }

// func forwardSocks(proxy *ProxyInfo, r *http.Request, clientConn net.Conn, done chan bool) *netutils.CounterConn {

// 	cl := socks5.Client{
// 		Auth: &netutils.UserInfo{
// 			User: proxy.User,
// 			Pass: proxy.Pass,
// 		},
// 		Timeout: 2 * time.Second,
// 	}
// 	destConn, err := cl.Open(proxy.Addr())
// 	if err != nil {
// 		debug(999, "Failed upstream socks", err)
// 		done <- true
// 		return nil
// 	}

// 	go func() {
// 		defer func() { done <- true }()

// 		hst, port := netutils.GetHostPort(r)
// 		err = destConn.Connect(hst, port)
// 		if err != nil {
// 			debug(999, "Failed upstream socks[connect]", err)
// 			return
// 		}

// 		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
// 		defer cancel()
// 		nr := r.Clone(ctx)

// 		if len(proxy.User) != 0 || len(proxy.Pass) != 0 {
// 			basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(proxy.Auth()))
// 			nr.Header.Set("Proxy-Authorization", basicAuth)
// 		} else {
// 			nr.Header.Del("Proxy-Authorization")
// 		}

// 		// reader := bufio.NewReader(destConn)
// 		// Write the Request to the Upstream Proxy
// 		nr.WriteProxy(destConn)
// 		go func() {
// 			io.Copy(destConn, clientConn)
// 			destConn.Close()
// 			clientConn.Close()
// 		}()
// 		io.Copy(clientConn, destConn)
// 		destConn.Close()
// 		clientConn.Close()
// 	}()
// 	return destConn.Conn.(*netutils.CounterConn)
// }

// IsExpired returns true if ExireAt is true and is in the past
func (s *Server) IsExpired() bool {
	if s.ExpireAt == nil {
		return false
	}
	if s.ExpireAt.Before(time.Now()) {
		return true
	}
	return false
}

// SelectProxy finds a matching proxy for this request
func (s *Server) SelectProxy(uinfo *netutils.UserInfo, m map[string]string) (*ProxyInfo, error) {

	var proxyP *ProxyInfo

	m["Group"] = "Default"

	// parses only the group to select the right proxy
	ParseProxyParams(&m, true)

	m["Group"] = strings.ToUpper(m["Group"])

	if s.Devices != nil {
		// if requested specific device
		d, ok := m["device"]
		if !ok {
			for v := range s.Devices {
				d = v
				break
			}
			m["device"] = d
		}
		proxyP = getDynamicProxy(d)

		if proxyP == nil {
			return nil, errors.New("No such proxy")
		}
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
		return nil, errors.New("Could not find proxy")
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

	return proxy, nil
}

// RunAccountant manages bandwidth counting
func (s *Server) RunAccountant(tp string, downstream net.Conn, upstream net.Conn) {

	ds := downstream.(*netutils.CounterConn)

	us := upstream.(*netutils.CounterConn)

	if us.Upstream+us.Downstream > ds.Upstream+ds.Downstream {
		s.LogConnection(s.Addr, tp, us.Downstream, us.Upstream)
		s.Consume(us.Downstream, us.Upstream)
	} else {
		s.LogConnection(s.Addr, tp, ds.Downstream, ds.Upstream)
		s.Consume(ds.Downstream, ds.Upstream)
	}
}

// RunPiper pipes data between upstream and downstream and closes one when the other closes
func (s *Server) RunPiper(downstream net.Conn, upstream net.Conn) {

	// pipe content
	go transfer(downstream, upstream)
	transfer(upstream, downstream)
}

// transfer just copies from source to destination then closes both
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

// CheckAuth checks if the provided user information (user, password, ip) matches
// the server and returns and returns an error if it does not
func (s *Server) CheckAuth(uinfo *netutils.UserInfo, ip string) error {
	if s.Auth.Type == AuthTypeUserPass {
		if CheckUser(uinfo, s.Auth) == false {
			debug(99, fmt.Sprintf("[Socks](%s)", s.Addr),
				"Wrong User/Pass Combination:",
				"\n\tActual:", uinfo.User, uinfo.Pass,
				"\n\tExpected", s.Auth.User, s.Auth.Pass)

			return errors.New("Invalid User or Password")
		}
	} else {
		if val, ok := s.Auth.IP[ip]; !ok || val == false {
			debug(999, fmt.Sprintf("[Socks](%s) IP not allowed: %s", s.Addr, ip))

			return errors.New("IP not allowed")
		}
	}
	return nil
}
