package httpproxy

import (
	"io"
	"net/http"
	"strings"
	"time"
)

func (proxy *Proxy) handleHTTP(req *http.Request, conn io.ReadWriteCloser) {
	for k := range req.Header {
		low := strings.ToLower(k)
		if strings.HasPrefix(low, "proxy") {
			req.Header.Del(k)
		}
	}
	var netTransport = &http.Transport{
		Dial: proxy.Forwarder.Forward,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	resp, err := netTransport.RoundTrip(req)
	if err != nil {
		// http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	resp.Write(conn)

	io.Copy(conn, resp.Body)
}
