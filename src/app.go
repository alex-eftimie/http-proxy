// "alexeftimie:tTzmG0lgVj5HT7fY@proxy.packetstream.io:31112"
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	// Run the Master Controller
	Run()

	log.Println("Running servers")
	for _, server := range Ca.Servers {
		RunServer(server)
	}

	quit := BandwidthMonitor()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for {
			<-sigs

			fmt.Println("")
			quit <- true
			CloseServers()
			Ca.Sync()
			os.Exit(0)
		}
	}()

	// go RunController()

	for {
		time.Sleep(1 * time.Second)
	}
}
