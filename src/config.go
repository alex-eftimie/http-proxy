package main

import (
	"log"
	"time"

	apiconfig "github.com/Alex-Eftimie/api-config"
	"github.com/fatih/color"
)

// ConfigType Stores all the configuration options and session
type ConfigType struct {
	apiconfig.Configuration
	DebugLevel              int
	BindAddr                string
	ServerIP                string
	BandwidthUpdateInterval uint
	Proxies                 []string
	ProxyAgent              string
	SessionTimeout          time.Duration
}

// CacheType stores Servers
type CacheType struct {
	apiconfig.Configuration
	ServerPort int
	Servers    []*Server
	Proxies    []*HTTPProxy       `json:"-"`
	ServerMap  map[string]*Server `json:"-"`
}

// Co is the main config object
var Co *ConfigType

// Ca is the main cache object
var Ca *CacheType

func init() {
	log.Println("Reading Config")
	Co = &ConfigType{
		Configuration: *apiconfig.NewConfig("config/config.jsonc"),
	}

	Ca = &CacheType{
		Configuration: *apiconfig.NewConfig("cache/cache.json"),
	}

	Co.LoadConfig(Co)
	Ca.LoadConfig(Ca)

	// turn nanoseconds into seconds
	Co.SessionTimeout *= time.Second

	Ca.Lock()
	Ca.ServerMap = make(map[string]*Server)
	Ca.Proxies = make([]*HTTPProxy, 0)

	for _, v := range Co.Proxies {
		hp := ReadProxy(v)
		Ca.Proxies = append(Ca.Proxies, hp)
	}
	Ca.Unlock()

	// go func() {
	// 	time.Sleep(30 * time.Second)

	// 	// TODO: change this, it will get very slow
	// 	Ca.Sync()
	// 	log.Println("Synced messages")
	// }()
	log.Println("Done reading config")
}

// BandwidthMonitor runs periodically and syncs the Cache File
func BandwidthMonitor() chan bool {
	dur := time.Duration(Co.BandwidthUpdateInterval)
	ticker := time.NewTicker(dur * time.Second)
	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println(color.RedString("Updating bandwidth"))
				Ca.Sync()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	return quit
}
