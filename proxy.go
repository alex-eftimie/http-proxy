package httpproxy

import (
	"log"
	"net"
	"time"

	"github.com/fatih/color"
)

// Proxy holds the config for a http/https proxy instance
type Proxy struct {
	BindAddr          string
	AuthCallback      func(user, password, ip string) bool
	BandwidthCallback func(user string, upload, download int64) bool
}

func (proxy *Proxy) Run() {

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
			proxy.HandleClient(c)
		}
	}()

	for {
		time.Sleep(time.Second * 10)
	}

}
