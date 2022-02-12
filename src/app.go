// "alexeftimie:tTzmG0lgVj5HT7fY@proxy.packetstream.io:31112"
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	apiconfig "github.com/alex-eftimie/api-config"
)

var flagFilter Filter

var cmd = flag.String("cmd", "", "Command to run")
var sortKey = flag.String("sort", "connections", "The key to sort logs by\nPossible values: connections, upstream, downstream, bandwidth")
var flagHuman = flag.Bool("h", false, "Human readable format")

func main() {
	os.Setenv("TZ", "UTC")
	loadConfig()
	loadLogger()
	loadSessions()
	flag.Var(&flagFilter, "filter", "The filter to run against the logs sorting")
	flag.Parse()

	switch *cmd {
	case "log:parse":
		logParse()
		os.Exit(0)
	}

	log.Println("Running Controller")
	// Run the Master Controller
	Run()

	// run the dynamic proxy router
	initDynamic()

	//
	// go runJugglerClient()

	log.Println("Running servers")
	for _, server := range Ca.Servers {
		err := RunServer(server)
		if err != nil {
			log.Fatalln("RunServer Error", err)
		}
	}
	event("ServersStarted")

	quit := BandwidthMonitor()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for {
			<-sigs
			go func() {
				time.Sleep(10 * time.Second)
				log.Fatalln("Shutdown failed to finish in a timely maner")
			}()
			log.Println("\nInitializing shutdown sequence")
			quit <- true
			CloseServers()
			// Ca.Sync()
			log.Println("Syncing cache")
			apiconfig.Sync(Ca)
			event("ServersStopped")

			log.Println("Finished shutdown sequence")
			os.Exit(0)
		}
	}()

	// go RunController()

	for {
		time.Sleep(1 * time.Second)
	}
}
