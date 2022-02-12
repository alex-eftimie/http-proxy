package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alex-eftimie/netutils"
	"github.com/alex-eftimie/utils"
	"github.com/davecgh/go-spew/spew"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	proxyproto "github.com/pires/go-proxyproto"
	"github.com/shenwei356/util/bytesize"
	"github.com/soheilhy/cmux"
)

// Accountable is an interface for net.Conn that contain a CounterConn
type Accountable interface {
	GetCounterConn() *netutils.CounterConn
}

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

func loadLogger() {

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
	H2CServer      *H2CServer          `json:"-"`
	CMux           cmux.CMux           `json:"-"`
	Client         string
	Addr           string
	Auth           Auth
	Bytes          *PrettyByte
	MaxThreads     *int
	MaxIPs         *int
	DefaultGroup   *string `json:",omitempty"`
	ExpireAt       *utils.Time
	parent         *CacheType // don't export it, it will cause cycles
	limiter        *ServerLimiter
	Devices        map[string]string      `json:",omitempty"`
	BWUsageHistory map[time.Time]*BWUsage `json:",omitempty"`
	DeviceSlice    []string               `json:"-"`
}

// RunServer starts a http listener on s.Addr
func RunServer(s *Server) error {
	Ca.Lock()
	defer Ca.Unlock()
	if s.Auth.Type != AuthTypeIP && s.Auth.Type != AuthTypeUserPass {
		log.Fatalf("Server %s has invalid Auth.Type %s", s.ID, s.Auth.Type)
	}

	// check for ServerMap collisions
	if _, ok := Ca.ServerMap[s.ID]; ok {
		return fmt.Errorf("Server id %s already exists", s.ID)
	}
	if _, ok := Ca.ServerMap[s.Auth.AuthToken]; ok {
		return fmt.Errorf("Server AuthToken %s already exists", s.Auth.AuthToken)
	}
	if _, ok := Ca.ServerMap[s.Addr]; ok {
		return fmt.Errorf("Server Addr %s already exists", s.Addr)
	}

	s.parent = Ca

	Ca.ServerMap[s.ID] = s

	Ca.ServerMap[s.Auth.AuthToken] = s

	var list []string
	for k := range s.Devices {
		list = append(list, k)
	}
	s.DeviceSlice = list

	Ca.ServerMap[s.Addr] = s

	s.SyncBandwidth()

	if len(s.BWUsageHistory) == 0 {
		s.BWUsageHistory = make(map[time.Time]*BWUsage)
	}

	// bind on the serverIP if the ip part of the addr is empty
	addr := s.Addr
	if addr[0] == ':' {
		addr = Co.ServerIP + addr
	}

	// test port first
	// Create the main listener.
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	proxyListener := &proxyproto.Listener{
		Listener: l,
		Policy:   proxyproto.MustLaxWhiteListPolicy(Co.AllowProxyProtocolFrom),
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
	s.H2CServer = newH2CServer(s)

	// return ListenAndServe(hs)
	go s.Serve(proxyListener)
	return nil
}

func (s *Server) Close() {
	// s.CMux.Close()
	if s.HTTPServer == nil {
		spew.Config.MaxDepth = 1
		spew.Dump(s.Addr, s)
	}
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
	h2cMatcher := s.CMux.Match(cmux.HTTP1Fast("PRI"))

	// run the servers
	go s.HTTPServer.Serve(httpMatcher)
	go s.Socks5Server.Serve(socks5Matcher)
	go s.H2CServer.Serve(h2cMatcher)

	s.CMux.Serve()

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
func (s *Server) GetProxyAndTunnel(uinfo *netutils.UserInfo, m map[string]string, host string, port int) (net.Conn, *ProxyInfo, error) {

	proxy, err := s.SelectProxy(uinfo, m)

	if proxy == nil || err != nil {
		return nil, nil, errors.New("Proxy Offline")
	}

	tunnel, err := proxy.GetTunnel(host, port)
	return tunnel, proxy, err

}

// SelectProxy finds a matching proxy for this request
func (s *Server) SelectProxy(uinfo *netutils.UserInfo, m map[string]string) (*ProxyInfo, error) {

	var proxyP *ProxyInfo

	if s.DefaultGroup != nil {
		m["Group"] = *s.DefaultGroup
	} else {
		m["Group"] = "Default"
	}

	// parses only the group to select the right proxy
	ParseProxyParams(&m, true)

	m["Group"] = strings.ToUpper(m["Group"])

	saveSession := false

	key := ""
	// Are we under sticky conditions?
	if v, ok := m["sticky"]; ok {
		key = uinfo.User + "::" + v
		proxyP = SessionMaster.GetSession(key)
		if proxyP != nil {
			proxyP.Connect()
		} else {
			saveSession = true
		}
	}

	// we don't have a sticky proxy
	if proxyP == nil {

		// do we have a dedicated device array?
		if s.Devices != nil {
			// if requested specific device
			d, ok := m["device"]
			if !ok {

				// select a truly random key from the device slice
				d = s.DeviceSlice[rand.Intn(len(s.DeviceSlice))]

				m["device"] = d
			}
			typ, ok := s.Devices[d]
			if !ok {
				return nil, errors.New("No such proxy")
			}
			proxyP = getDynamicProxy(d, typ)

			if proxyP == nil {
				return nil, errors.New("No such proxy")
			}

			// static proxies do not use the accelerator ( pooled connections )
			// TODO: we could start to pool for a while during periods of activity
			proxyP.Connect()

		} else { // we're running on the public pool

			proxyP = GetRandomProxy(m["Group"])
		}
		// else if v, ok := m["sticky"]; ok { // IF sticky ip requested, get the same proxy
		// 	key := uinfo.User + "::" + v

		// 	proxyP = SessionMaster.GetSession(key)
		// 	if proxyP == nil {
		// 		// log.Println("set", key)
		// 		proxyP = GetRandomProxy(m["Group"])

		// 		// we're storing a cloned pi because we'll edit this one
		// 		SessionMaster.SetSession(key, proxyP.Clone())
		// 	} else {
		// 		// proxy pooling is disabled for sticky ips
		// 		// this is because it would be a nightmare to actually implement
		// 		proxyP.Connect()
		// 	}
	}
	if proxyP == nil {
		return nil, errors.New("Could not find proxy")
	}

	// we're under sticky so we save it
	if saveSession {
		SessionMaster.SetSession(key, proxyP.Clone())
	}

	m["User"] = proxyP.User
	m["Pass"] = proxyP.Pass

	// parse the rest too
	ParseProxyParams(&m, false)
	proxyP.User = m["User"]
	proxyP.Pass = m["Pass"]

	return proxyP, nil
}

// RunAccountant manages bandwidth counting
// bandwidth counting could theoreticaly break at:
// 1. Connection not set as CounterConn ✅
// 2. Accountant does not run on connection ✅
// 3.
func (s *Server) RunAccountant(tp string, downstream net.Conn, upstream net.Conn) {

	// if Co.DebugLevel > 99 {
	// 	str := reflect.TypeOf(downstream).String()
	// 	if str != "*netutils.CounterConn" {
	// 		reportError("RunAccountant downstream not *netutils.CounterConn but" + str)
	// 	}
	// 	str = reflect.TypeOf(upstream).String()
	// 	if str != "*netutils.CounterConn" {
	// 		reportError("RunAccountant upstream not *netutils.CounterConn but" + str)
	// 	}
	// }

	us := upstream.(*netutils.CounterConn)

	// h2c connections don't really have a net.Conn as downstream
	// so we only take into account the amount of bandwidth that we send to the upstream
	// proxies
	if downstream == nil {
		us.Downstream = us.Downstream * Co.BandwidthCounterOverflowPerMille / 1000
		us.Upstream = us.Upstream * Co.BandwidthCounterOverflowPerMille / 1000
		s.LogConnection(s.Addr, tp, us.Downstream, us.Upstream)
		s.Consume(us.Downstream, us.Upstream)
		us.Upstream = -1
		us.Downstream = -1
		return
	}

	ds := downstream.(*netutils.CounterConn)

	us.Downstream = us.Downstream * Co.BandwidthCounterOverflowPerMille / 1000
	us.Upstream = us.Upstream * Co.BandwidthCounterOverflowPerMille / 1000

	ds.Downstream = ds.Downstream * Co.BandwidthCounterOverflowPerMille / 1000
	ds.Upstream = ds.Upstream * Co.BandwidthCounterOverflowPerMille / 1000

	if us.Upstream+us.Downstream > ds.Upstream+ds.Downstream {
		s.LogConnection(s.Addr, tp, us.Downstream, us.Upstream)
		s.Consume(us.Downstream, us.Upstream)
	} else {
		s.LogConnection(s.Addr, tp, ds.Downstream, ds.Upstream)
		s.Consume(ds.Downstream, ds.Upstream)
	}
	ds.Upstream = -1
	ds.Downstream = -1
	us.Upstream = -1
	us.Downstream = -1
}

// RunPiper pipes data between upstream and downstream and closes one when the other closes
func (s *Server) RunPiper(downstream io.ReadWriteCloser, upstream io.ReadWriteCloser) {

	// pipe content
	go transfer(downstream, upstream)
	transfer(upstream, downstream)
}

// transfer just copies from source to destination then closes both
func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
	utils.Debug(999, "Closing sockets")
}

// CheckAuth checks if the provided user information (user, password, ip) matches
// the server and returns and returns an error if it does not
func (s *Server) CheckAuth(uinfo *netutils.UserInfo, ip string) error {
	if s.Auth.Type == AuthTypeUserPass {
		if CheckUser(uinfo, s.Auth) == false {
			debug(99, fmt.Sprintf("[ALL](%s)", s.Addr),
				"Wrong User/Pass Combination:",
				"\n\tActual:", uinfo.User, uinfo.Pass,
				"\n\tExpected", s.Auth.User, s.Auth.Pass)

			return errors.New("Invalid User or Password")
		}
	} else {
		if val, ok := s.Auth.IP[ip]; !ok || val == false {
			debug(999, fmt.Sprintf("[ALL](%s) IP not allowed: %s", s.Addr, ip))

			return errors.New("IP not allowed")
		}
	}
	return nil
}
