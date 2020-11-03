package main

import (
	"log"

	httpproxy "github.com/alex-eftimie/http-proxy"
)

func main() {
	c := &httpproxy.Proxy{
		BindAddr: "0.0.0.0:998",
		AuthCallback: func(user, pass, ip string) bool {
			log.Printf("Authenticating %s, %s, %s, %t\n", user, pass, ip, true)
			return true
		},
	}

	c.Run()
}
