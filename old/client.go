package httpproxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/alex-eftimie/networkhelpers"
)

var errorResponse = "HTTP/1.0 400 Bad Request\r\n\r\nSomething went terribly wrong\n"
var successResponse = "HTTP/1.1 200 OK\r\n\r\n"
var authReqiredResponse = "HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"%s\"\r\n\r\n\n"

type proxyConn struct {
	*networkhelpers.CounterConn
	user string
	ip   string
}

// HandleClient runs all the logic of a proxy after the connection is established
func (proxy *Proxy) HandleConn(c io.ReadWriteCloser, ip string) {

	log.Println("New client")

	cc := networkhelpers.NewCounterConn(c)
	pc := &proxyConn{cc, "", ip}
	proxy.handleClient(pc)

	if proxy.BandwidthCallback != nil {
		proxy.BandwidthCallback(pc.user, pc.ip, pc.Counter.Upstream, pc.Counter.Downstream)
	}
}

func (proxy *Proxy) handleClient(conn *proxyConn) {
	var auth string
	reader := bufio.NewReader(conn)

	req, err := http.ReadRequest(reader)
	log.Println("Request read")
	if err != nil {
		log.Println("HTTP error:", err)
	} else {
		auth = req.Header.Get("Proxy-Authorization")
	}

	if proxy.AuthCallback != nil {

		if len(auth) == 0 {
			io.WriteString(conn, fmt.Sprintf(authReqiredResponse, proxy.Realm))
			conn.Close()
			return
		}

		parts := strings.Split(auth, " ")

		if len(parts) != 2 || strings.ToLower(parts[0]) != "basic" {

			io.WriteString(conn, errorResponse)
			conn.Close()
			return
		}

		sDec, _ := base64.StdEncoding.DecodeString(parts[1])
		auth = string(sDec)
		parts = strings.Split(auth, ":")

		conn.user = parts[0]

		if len(parts) != 2 || false == proxy.AuthCallback(parts[0], parts[1], conn.ip) {
			io.WriteString(conn, fmt.Sprintf(authReqiredResponse, proxy.Realm))
			conn.Close()
			return
		}
	}

	if req.Method == http.MethodConnect {
		proxy.handleTunnel(req, conn)
		return
	}
	proxy.handleHTTP(req, conn)

}
