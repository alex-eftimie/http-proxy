# http-proxy

You can find an examples  in /examples/
```go

func main() {
	c := &httpproxy.Proxy{
		BindAddr: "0.0.0.0:998",
		AuthCallback: func(user, pass, ip string) bool {
			log.Printf("Authenticating %s, %s, %s, %t\n", user, pass, ip, true)
			return true
		},
		BandwidthCallback: func(user string, upload, download int64) {
			log.Printf("BWC Upload: %s, Download: %d, %d\n", user, upload, download)
		},
	}

	c.Run()
}

```