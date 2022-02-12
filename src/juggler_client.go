package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/alex-eftimie/netutils"
	"github.com/gorilla/websocket"
)

var writer chan interface{}

var dynMux sync.RWMutex
var dyn map[string]*ProxyInfo

func initDynamic() {
	dyn = make(map[string]*ProxyInfo)
}

func getDynamicProxy(deviceIndex string, typ string) *ProxyInfo {

	list := strings.SplitN(typ, ":", 2)
	mapKey := list[1]

	// read the dynamic proxy map
	dynMux.RLock()
	did, ok := dyn[mapKey]
	dynMux.RUnlock()

	// we haven't found this proxy
	if !ok {

		// if we're using a static proxy definition then it's not yet loaded
		if list[0] == "staticProxy" {

			// parse the proxy info into a *ProxyInfo
			did = ReadProxy(mapKey)

			// lock and update the dynamic proxy map
			dynMux.Lock()
			dyn[mapKey] = did
			dynMux.Unlock()
		} else {
			return nil
		}
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

	writer = make(chan interface{})

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
		n := &netutils.Node{}
		for {
			err := c.ReadJSON(n)
			if err != nil {
				log.Println("read:", err)
				return
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
			log.Println("Host update", n.UUID, n.Host, n.Port)
		}
	}()

	for {
		select {
		case msg := <-writer:
			err := c.WriteJSON(msg)
			// err := c.WriteMessage(websocket.BinaryMessage, msg)
			if err != nil {
				log.Println("write:", err)
				return
			}
		}
	}

}
