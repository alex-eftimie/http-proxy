package main

import (
	"log"
	"net"
	"time"

	httpproxy "github.com/Alex-Eftimie/http-proxy"
	"github.com/Alex-Eftimie/networkhelpers"
	"github.com/fatih/color"
)

func main() {
	p := &httpproxy.Proxy{
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

	Run(p)
}

func Run(proxy *httpproxy.Proxy) {
	l, err := net.Listen("tcp4", proxy.BindAddr)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Client Handler", color.YellowString("listening"), "on", color.MagentaString(proxy.BindAddr))

	go func() {
		defer l.Close()

		for {
			c, err := l.Accept()
			if err != nil {
				log.Println("clientHandler:", color.RedString(err.Error()))
				return
			}
			ip := networkhelpers.RemoteAddr(c)
			proxy.HandleConn(c, ip)
		}
	}()

	for {
		time.Sleep(time.Second * 10)
	}
}
