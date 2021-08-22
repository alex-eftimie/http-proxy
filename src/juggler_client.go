package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"

	"github.com/Alex-Eftimie/netutils"
	"github.com/gorilla/websocket"
)

var writer chan []byte

var dyn map[string]*ProxyInfo

func initDynamic() {
	dyn = make(map[string]*ProxyInfo)
}

func getDynamicProxy(deviceID string) *ProxyInfo {
	did, ok := dyn[deviceID]
	if !ok {
		return nil
	}
	return did
}

func notifyJuggler(n *netutils.Node) {
	writer <- n.JSON()
}
func runJugglerClient() {

	if Co.JugglerAddr == "" || Co.JugglerAuthToken == "" {
		return
	}

	writer = make(chan []byte)

	u := url.URL{Scheme: "ws", Host: Co.JugglerAddr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	h := make(http.Header)
	h.Add("Authorization", "Bearer "+Co.JugglerAuthToken)

	c, resp, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err != nil {
		if resp != nil {
			log.Fatal("dial:", err, resp.StatusCode)
		}
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)

			n := &netutils.Node{}
			err = json.Unmarshal(message, n)
			if err != nil {
				log.Println("dynproxyError")
				continue
			}

			if n.Host == "" {
				delete(dyn, n.UUID)
				continue
			}

			dyn[n.UUID] = &ProxyInfo{
				Host: n.Host,
				Port: n.Port,
				Type: TypeSocks5,
			}
		}
	}()

	for {
		select {
		case msg := <-writer:
			err := c.WriteMessage(websocket.BinaryMessage, msg)
			if err != nil {
				log.Println("write:", err)
				return
			}
		}
	}

}
