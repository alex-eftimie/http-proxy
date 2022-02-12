package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"

	apiconfig "github.com/alex-eftimie/api-config"
)

// CacheType Stores all the configuration options and session
type CacheType struct {
	apiconfig.Configuration
	Servers []*NanoServer
}

// NanoServer is a single Proxy
type NanoServer struct {
	Addr string
}

var template string = `server {
    listen {{ip}}:{{port}};
    proxy_pass 127.0.0.1:{{port}};
    proxy_protocol on;
}

`

// Co is the main config object
var Co *CacheType

// flag vars
var (
	cols string
	cmd  string
	sep  string
	prnt bool
)

func flags() {

	flag.StringVar(&cols, "cols", "[1,2]", "columns to count, json string")
	flag.StringVar(&cmd, "cmd", "configure", "comand to run, configure(default) | count")
	flag.StringVar(&sep, "sep", " ", "separator to use for splitting default: single space")
	flag.BoolVar(&prnt, "print", true, "print parsed lines? default: true")
	flag.Parse()
}

func main() {
	flags()
	if cmd == "count" {
		count()
		os.Exit(0)
	} else if cmd != "configure" {
		log.Fatalln("Invalid command", cmd)
	}

	// log.Println("Reading Config")
	Co = &CacheType{
		Configuration: *apiconfig.NewConfig("cache/cache.json"),
	}
	apiconfig.LoadConfig(Co)
	msg := ""
	for _, v := range Co.Servers {
		s := strings.Replace(template, "{{port}}", v.Addr[1:], -1)
		s = strings.Replace(s, "{{ip}}", os.Args[2], -1)
		msg += s
	}
	// spew.Dump(Co)
	// fmt.Println(msg)
	d1 := []byte(msg)
	err := os.WriteFile(os.Args[3], d1, 0755)
	if err != nil {
		log.Fatalln(err)
	}

	cmd := exec.Command("bash", "-c", os.Args[4])
	o, err := cmd.Output()
	if err != nil {
		log.Fatalln(string(o))
	}
}
