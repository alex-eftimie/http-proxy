package main

import (
	"log"

	hf "github.com/Alex-Eftimie/http-forwarder"
	httpproxy "github.com/Alex-Eftimie/http-proxy"
	tf "github.com/Alex-Eftimie/tcp-forwarder"
)

func main() {

	use_tcp := false

	var fwd httpproxy.Forwarder
	if use_tcp {
		fwd = tf.NewForwarder()
	} else {
		fwd = hf.NewForwarder("Alexeftimie:tTzmG0lgVj5HT7fY@proxy.packetstream.io:31112")
	}

	c := &httpproxy.Proxy{
		Realm:    "My Awesome Proxy Server",
		BindAddr: "0.0.0.0:998",
		AuthCallback: func(user, pass, ip string) bool {
			log.Printf("Authenticating %s, %s, %s, %t\n", user, pass, ip, true)

			// true = allow, false = disallow
			return true
		},
		BandwidthCallback: func(user, ip string, upload, download int64) {
			log.Printf("[BWC] %s(%s) Upload:%d, Download: %d\n", user, ip, upload, download)
		},
		// Forwarder: nil
		Forwarder: fwd,
	}

	c.Run()
}
