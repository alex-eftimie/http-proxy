package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
)

// HTTPProxy forwards represents a http proxy
type HTTPProxy struct {
	User string
	Pass string
	Host string
	Port int
}

// ReadProxy returns a pointer to a new instance of HTTPProxy parsed from proxy
// @param proxy string format user:pass@ip:port
func ReadProxy(proxy string) *HTTPProxy {

	r, _ := regexp.Compile("^((?P<user>[^:@]+)(:(?P<pass>[^:@]+)){0,}@){0,1}(?P<host>[0-9A-Za-z\\.\\-]+):(?P<port>[0-9]+)")
	res := r.FindStringSubmatch(proxy)
	result := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = res[i]
		}
	}

	port, _ := strconv.Atoi(result["port"])
	return &HTTPProxy{
		User: result["user"],
		Pass: result["pass"],
		Host: result["host"],
		Port: port,
	}
}

// GetRandomProxy returns a random proxy from the Proxy Cache
func GetRandomProxy() *HTTPProxy {

	proxies := len(Ca.Proxies)
	if proxies == 1 {
		return Ca.Proxies[0]
	}

	randomIndex := rand.Intn(proxies)

	return Ca.Proxies[randomIndex]
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
			m[ss[0]] = ss[1]
		}
	}
	return ret
}

// Addr returns a string Host:Port
func (hp *HTTPProxy) Addr() string {
	return fmt.Sprintf("%s:%d", hp.Host, hp.Port)
}

// Auth returns a string User:Pass
func (hp *HTTPProxy) Auth() string {
	return fmt.Sprintf("%s:%s", hp.User, hp.Pass)
}
