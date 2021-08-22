package main

import (
	"log"
	"time"

	apiconfig "github.com/Alex-Eftimie/api-config"
	"github.com/Alex-Eftimie/utils"
	"github.com/fatih/color"
)

// ConfigType Stores all the configuration options and session
type ConfigType struct {
	apiconfig.Configuration
	DebugLevel              int
	BindAddr                string
	ServerIP                string
	BandwidthUpdateInterval uint
	Proxies                 map[string][]string
	ProxyAgent              string
	LogParserSortKey        int
	DomainMapping           map[string][]string
	SessionTimeout          time.Duration
	LogRequestsLevel        int
	JugglerAddr             string
	JugglerAuthToken        string
}

// CacheType stores Servers
type CacheType struct {
	apiconfig.Configuration
	ServerPort int
	Servers    []*Server
	Proxies    map[string][]*ProxyInfo `json:"-"`
	ServerMap  map[string]*Server      `json:"-"`
	DomainMap  map[string]string       `json:"-"`
}

// func (s *CacheType) Lock() {
// 	log.Println("Locking Cache")
// 	s.Configuration.Lock()
// }
// func (s *CacheType) Unlock() {
// 	log.Println("Unlocking Cache")
// 	s.Configuration.Unlock()
// }

// Co is the main config object
var Co *ConfigType

// Ca is the main cache object
var Ca *CacheType

func init() {
	// log.Println("Reading Config")
	Co = &ConfigType{
		Configuration: *apiconfig.NewConfig("config/config.jsonc"),
	}

	Ca = &CacheType{
		Configuration: *apiconfig.NewConfig("cache/cache.json"),
	}

	apiconfig.LoadConfig(Co)
	apiconfig.LoadConfig(Ca)

	// turn nanoseconds into seconds
	Co.SessionTimeout *= time.Second

	Ca.Lock()
	Ca.ServerMap = make(map[string]*Server)
	Ca.Proxies = make(map[string][]*ProxyInfo, 0)

	for group, arr := range Co.Proxies {
		Ca.Proxies[group] = make([]*ProxyInfo, 0)
		for _, proxyStr := range arr {
			hp := ReadProxy(proxyStr)
			Ca.Proxies[group] = append(Ca.Proxies[group], hp)
		}
	}
	Ca.DomainMap = make(map[string]string, len(Co.DomainMapping))

	// dst is actual domain ( google.com, facebook.com )
	// dlist is a list of domains controlled by them google-analytics, fbcdn, etc...
	for dst, dlist := range Co.DomainMapping {
		for _, src := range dlist {
			Ca.DomainMap[src] = dst
		}
	}
	Ca.Unlock()

	// go func() {
	// 	time.Sleep(30 * time.Second)

	// 	// TODO: change this, it will get very slow
	// 	Ca.Sync()
	// 	log.Println("Synced messages")
	// }()
	// log.Println("Done reading config")
	utils.DebugLevel = Co.DebugLevel
}

// func (ca *CacheType) Lock() {
// 	log.Println("Lock cache")
// 	ca.Configuration.Lock()
// }
// func (ca *CacheType) Unlock() {
// 	log.Println("Unlock cache")
// 	ca.Configuration.Unlock()
// }

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
				// Ca.Sync()
				Ca.Lock()
				for _, srv := range Ca.Servers {
					if len(srv.BWUsageHistory) > 31 {
						// TODO: Improve, check only required dates
						now := time.Now()
						for t := range srv.BWUsageHistory {
							if now.Sub(t).Hours()/24 > 31 {
								log.Println("Deleting bw,", t)
								delete(srv.BWUsageHistory, t)
							}
						}
					}
				}
				Ca.Unlock()
				apiconfig.Sync(Ca)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	return quit
}
