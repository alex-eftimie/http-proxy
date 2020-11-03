package main

import (
	"log"

	httpproxy "github.com/alex-eftimie/http-proxy"
)

func main() {
	c := &httpproxy.Proxy{
		Realm:    "My Awesome Proxy Server",
		BindAddr: "0.0.0.0:998",
		AuthCallback: func(user, pass, ip string) bool {
			log.Printf("Authenticating %s, %s, %s, %t\n", user, pass, ip, true)
			return true
		},
		BandwidthCallback: func(user, ip string, upload, download int64) {
			log.Printf("[BWC] %s(%s) Upload:%d, Download: %d\n", user, ip, upload, download)
		},
	}

	c.Run()
}
