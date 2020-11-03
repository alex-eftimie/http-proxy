package httpproxy

import (
	"io"
	"log"
	"net"
	"net/http"

	"github.com/fatih/color"

	nh "github.com/alex-eftimie/networkhelpers"
)

func handleTunnel(req *http.Request, conn net.Conn) {

	addr := req.Host

	upstream := nh.ConnectTCP(addr)
	log.Println("Proxy to", color.MagentaString(addr))

	// failed to connect
	if upstream == nil {
		log.Println("Failed to connect to the upstream")
		conn.Close()
		return
	}

	io.WriteString(conn, successResponse)

	nh.PipeStreams(conn, upstream)

}
