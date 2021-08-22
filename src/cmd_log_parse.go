package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/shenwei356/util/bytesize"
)

type LogLine struct {
	DateTime   time.Time
	Type       string
	ServerID   string
	ServerAddr string
	Host       string
	_          string
	Upload     int64
	_          string
	Download   int64
}

type Info struct {
	Domain      string
	Connections int64
	Upstream    int64
	Downstream  int64
}

//  Len() int
//  Less (i , j) bool
//  Swap(i , j int)

func (s Smap) Len() int {
	return len(s.keys)
}
func (s Smap) Less(i, j int) bool {
	if s.SortKey == 0 {
		return s.Datamap[s.keys[i]].Connections > s.Datamap[s.keys[j]].Connections
	}
	if s.SortKey == 1 {
		return s.Datamap[s.keys[i]].Upstream > s.Datamap[s.keys[j]].Upstream
	}
	if s.SortKey == 2 {
		return s.Datamap[s.keys[i]].Downstream > s.Datamap[s.keys[j]].Downstream
	}
	if s.SortKey == 3 {
		return s.Datamap[s.keys[i]].Downstream+s.Datamap[s.keys[i]].Upstream >
			s.Datamap[s.keys[j]].Downstream+s.Datamap[s.keys[j]].Upstream
	}
	log.Fatal("Invalid LogParserSortKey")
	return false
}
func (s Smap) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	//[i], s[j] = s[j], s[i]
}

// Smap is a map that can be sorted by it's items keys
// 0 sorts by connection count
// 1 sorts by upload byte count
// 2 sorts by download byte count
// 3 sorts by upload+download byte count
type Smap struct {
	Datamap map[string]*Info
	keys    []string
	SortKey int
}

type Filter struct {
	M           map[string]string
	HasType     bool
	HasServerID bool
	HasDomain   bool
}

func (f *Filter) String() string {
	return spew.Sdump(f.M)
}
func (f *Filter) Set(value string) error {
	f.M = make(map[string]string)
	spl := strings.Split(value, ",")
	for _, flt := range spl {
		parts := strings.Split(flt, ":")
		f.M[parts[0]] = parts[1]
		switch parts[0] {
		case "Type":
			f.HasType = true
		case "ServerID":
			f.HasServerID = true
		case "Domain":
			f.HasDomain = true
		}
	}
	return nil
}

func logParse() {

	data := Smap{
		Datamap: make(map[string]*Info),
		SortKey: Co.LogParserSortKey,
	}

	// log.Println("Parsing access log files")

	file, err := os.Open("logs/access.log")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		line := scanner.Text()
		parts := strings.Split(line, " ")

		upl, _ := strconv.ParseInt(parts[7], 10, 64)
		dl, _ := strconv.ParseInt(parts[9], 10, 64)
		dt, _ := time.Parse("2006/01/02 15:04:05", parts[0]+" "+parts[1])

		ll := &LogLine{
			DateTime:   dt,
			Type:       parts[2],
			ServerID:   parts[3],
			ServerAddr: parts[4],
			Host:       parts[5],
			// _:          parts[5],
			Upload: upl,
			// _:          parts[8],
			Download: dl,
		}
		// filter by id, by addr, by protocol, by domain, by any combination
		// if *flagFilter != "" {
		// 	// if *flagFilter != ll.ServerID && *flagFilter !=
		// }

		domain := parseDomain(ll.Host)

		if flagFilter.HasDomain && flagFilter.M["Domain"] != domain ||
			flagFilter.HasServerID && flagFilter.M["ServerID"] != ll.ServerID ||
			flagFilter.HasType && flagFilter.M["Type"] != ll.Type {
			continue
		}

		var mapKey = ll.ServerID + "::" + domain

		current, ok := data.Datamap[mapKey]
		if !ok {
			current = &Info{
				Domain: domain,
			}
			data.Datamap[mapKey] = current
			data.keys = append(data.keys, mapKey)
		}
		current.Connections++
		current.Downstream += ll.Download
		current.Upstream += ll.Upload

		// os.Exit(0)
	}

	sort.Sort(data)

	for _, v := range data.keys {
		item := data.Datamap[v]
		if *flagHuman {
			u := bytesize.ByteSize(item.Upstream)
			d := bytesize.ByteSize(item.Downstream)
			fmt.Printf("%s\t%d\t%s\t%s\n", item.Domain, item.Connections, u.String(), d.String())
		} else {
			fmt.Printf("%s\t%d\t%d\t%d\n", item.Domain, item.Connections, item.Upstream, item.Downstream)
		}
	}
	// spew.Dump(data.keys)

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func parseDomain(host string) string {
	parts := strings.Split(host, ".")
	l := len(parts)
	tld := parts[l-1]
	domain := parts[l-2]

	dn := ""
	switch domain {
	case "co", "com", "org":
		dn = parts[l-3] + "." + domain + "." + tld
	default:
		dn = domain + "." + tld
	}

	if mappedDomain, ok := Ca.DomainMap[dn]; ok {
		return mappedDomain
	}
	return dn
}
