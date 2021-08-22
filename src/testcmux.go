package main

// import (
// 	"io"
// 	"log"
// 	"net"
// 	"net/http"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"github.com/soheilhy/cmux"
// )

// type anyServer struct{}

// type socks5Server struct {
// }

// type Server struct {
// }

// func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	log.Println("http connection")
// }
// func (ss *socks5Server) ServeSocks(c net.Conn) {
// 	log.Println("Socks5 connection")
// }

// func (as *anyServer) Serve(list net.Listener) {
// 	log.Println("Any accept")
// 	for {
// 		rw, _ := list.Accept()
// 		log.Println("any listener")
// 		rw.Close()
// 	}
// }

// func (ss *socks5Server) Serve(list net.Listener) {

// 	log.Println("socks listen")
// 	var tempDelay time.Duration
// 	for {
// 		rw, err := list.Accept()
// 		log.Println("socks conn")

// 		if err != nil {

// 			if ne, ok := err.(net.Error); ok && ne.Temporary() {
// 				if tempDelay == 0 {
// 					tempDelay = 5 * time.Millisecond
// 				} else {
// 					tempDelay *= 2
// 				}
// 				if max := 1 * time.Second; tempDelay > max {
// 					tempDelay = max
// 				}
// 				log.Printf("http: Accept error: %v; retrying in %v", err, tempDelay)
// 				time.Sleep(tempDelay)
// 				continue
// 			}
// 			return
// 		}

// 		ss.ServeSocks(rw)
// 	}
// }

// func main() {
// 	NewDualServer(":1000")
// 	sigs := make(chan os.Signal, 1)
// 	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

// 	<-sigs
// }

// func NewDualServer(addr string) {

// 	// Create the main listener.
// 	l, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Println("Listening on", addr)

// 	m := cmux.New(l)
// 	socks5Matcher := m.Match(Socks5Matcher())
// 	httpMatcher := m.Match(cmux.HTTP1Fast())
// 	anyMatcher := m.Match(cmux.Any())

// 	socks5Server := &socks5Server{}
// 	httpServer := &http.Server{Handler: &Server{}}
// 	srv := anyServer{}

// 	go httpServer.Serve(httpMatcher)
// 	// go downstreamServer.Serve(downstreamMatcher)
// 	go socks5Server.Serve(socks5Matcher)
// 	go srv.Serve(anyMatcher)

// 	m.Serve()
// }

// func Socks5Matcher() cmux.Matcher {
// 	return func(r io.Reader) bool {
// 		b := make([]byte, 1)
// 		r.Read(b)
// 		return b[0] == 0x05
// 	}
// }
