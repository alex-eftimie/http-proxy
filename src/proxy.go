package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Alex-Eftimie/netutils"
	"github.com/Alex-Eftimie/socks5"
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
	User string
	Pass string
	Host string
	Port int
	Type proxyType
}

// GetTunnel connects to the proxy and establishes a tunnel
func (pi *ProxyInfo) GetTunnel(host string, port int) (net.Conn, error) {
	if pi.Type == TypeSocks5 {
		cl := &socks5.Client{
			Auth: &netutils.UserInfo{
				User: pi.User,
				Pass: pi.Pass,
			},
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
	dc, err := net.DialTimeout("tcp", pi.Addr(), 10*time.Second)
	if err != nil {
		return nil, err
	}
	destConn := &netutils.CounterConn{Conn: dc, Upstream: 0, Downstream: 0}
	host = fmt.Sprintf("%s:%d", host, port)
	nr := &http.Request{
		Method:     "CONNECT",
		ProtoMajor: 1,
		ProtoMinor: 1,
		URL: &url.URL{
			Host: host,
		},
		Host:   host,
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

	if err != nil || resp.StatusCode != http.StatusOK {
		destConn.Close()
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
			tp = "SOCKS5"
		} else {
			tp = "HTTP"
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

	proxies := len(Ca.Proxies[group])
	if proxies == 0 {
		return nil
	}
	if proxies == 1 {
		return Ca.Proxies[group][0]
	}

	randomIndex := rand.Intn(proxies)

	return Ca.Proxies[group][randomIndex]
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
func (hp *ProxyInfo) Addr() string {
	return fmt.Sprintf("%s:%d", hp.Host, hp.Port)
}

// Auth returns a string User:Pass
func (hp *ProxyInfo) Auth() string {
	return fmt.Sprintf("%s:%s", hp.User, hp.Pass)
}
