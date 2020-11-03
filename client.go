package httpproxy

import (
	"bufio"
	"encoding/base64"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/alex-eftimie/networkhelpers"
)

var errorResponse = "HTTP/1.0 400 Bad Request\r\n\r\nSomething went terribly wrong\n"
var successResponse = "HTTP/1.1 200 OK\r\n\r\n"
var authReqiredResponse = "HTTP/1.1 407 Proxy Authentication Required\r\nProxy-Authenticate: Basic realm=\"Freedom LOL\"\r\n\r\n\n"

// HandleClient Handles all the work for the http(s) proxy
func (proxy *Proxy) HandleClient(conn net.Conn) {

	log.Println("New client")

	var auth string

	reader := bufio.NewReader(conn)

	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("HTTP error:", err)
	} else {
		auth = req.Header.Get("Proxy-Authorization")
	}

	if proxy.AuthCallback != nil {

		ip := networkhelpers.RemoteAddr(conn)
		if len(auth) == 0 {
			io.WriteString(conn, authReqiredResponse)
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

		if len(parts) != 2 || false == proxy.AuthCallback(parts[0], parts[1], ip) {
			io.WriteString(conn, authReqiredResponse)
			conn.Close()
			return
		}
	}

	if req.Method == http.MethodConnect {
		handleTunnel(req, conn)
		return
	}
	handleHTTP(req, conn)

}
