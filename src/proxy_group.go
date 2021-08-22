package main

import (
	"fmt"
	"strings"

	apiconfig "github.com/Alex-Eftimie/api-config"
)

// G holds all the modifier groups
var G *Groups

// Groups is a struct representing the groups.jsonc file format
type Groups struct {
	apiconfig.Configuration
	Modifiers map[string]*ProxyGroup
}

// ProxyGroup is a single modifier
type ProxyGroup struct {
	Name   string
	Modify ProxyModify
	Map    map[string]string
}

// ProxyModify holds the new format for modifications for user and pass
type ProxyModify struct {
	User  *string
	Pass  *string
	Group *string
}

func init() {
	G = &Groups{
		Configuration: *apiconfig.NewConfig("./config/groups.jsonc"),
	}
	apiconfig.LoadConfig(G)
}

// ParseProxyParams receives a map of the _key-value pairs passed by client and
// runs all the modifications
func ParseProxyParams(mp *map[string]string, onlyGroup bool) {
	m := *mp
	for trigger, value := range m {
		if grp, ok := G.Modifiers[trigger]; ok {

			// first pass only needs to set group
			if onlyGroup && grp.Modify.Group == nil {
				continue
			}

			if v, ok := grp.Map[value]; ok {
				m[trigger] = v
			}
			if grp.Modify.User != nil {
				m["User"] = Replacer(*grp.Modify.User, m)
			}
			if grp.Modify.Pass != nil {
				m["Pass"] = Replacer(*grp.Modify.Pass, m)
			}
			if grp.Modify.Group != nil {
				m["Group"] = Replacer(*grp.Modify.Group, m)
			}
		}
	}
}

// Replacer replaces instances {param} with p[param] in the format parameter
func Replacer(format string, p map[string]string) string {
	args, i := make([]string, len(p)*2), 0
	for k, v := range p {
		args[i] = "{" + k + "}"
		args[i+1] = fmt.Sprint(v)
		i += 2
	}
	return strings.NewReplacer(args...).Replace(format)
}
