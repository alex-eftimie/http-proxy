package main

import (
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/Alex-Eftimie/netutils"
	"github.com/Alex-Eftimie/socks5"
	"github.com/Alex-Eftimie/utils"
	"github.com/soheilhy/cmux"
)

// Socks5Matcher helps cmux determine if a request is socks5
func Socks5Matcher() cmux.Matcher {
	return func(r io.Reader) bool {
		b := make([]byte, 1)
		r.Read(b)
		return b[0] == 0x05
	}
}

type CustomSocks5Server struct {
	*socks5.Server
	parent *Server
}

func newSocks5Server(s *Server) *CustomSocks5Server {
	ret := &CustomSocks5Server{
		parent: s,
	}
	ret.Server = &socks5.Server{
		AuthHandler:   ret.AuthHandler,
		TunnelHandler: ret.TunnelHandler,
	}
	return ret
}

// AuthHandler handles authentication for this socks5 Server
func (cs *CustomSocks5Server) AuthHandler(uinfo *netutils.UserInfo, ip string) bool {
	return true
}

// TunnelHandler handles tunneling for this socks5 Server
func (cs *CustomSocks5Server) TunnelHandler(uinfo *netutils.UserInfo, ip string, c net.Conn, upstreamHost string, upstreamPort int, sc socks5.StatusCallback) {

	if uinfo == nil {
		// sc("user-and-password-required.status", socks5.StatusConnectionNotAllowedByRuleset)
		// return
		uinfo = &netutils.UserInfo{}
	}

	m := make(map[string]string)

	if uinfo.User != "" {
		r := utils.ParseParams(uinfo.User, &m, false)
		if r != "" {
			uinfo.User = r
		}
	}

	if uinfo.Pass != "" {
		r := utils.ParseParams(uinfo.Pass, &m, false)
		// spew.Dump("XXXXXX", uinfo.Pass, r)
		if r != "" {
			uinfo.Pass = r
		}
	}

	if err := cs.parent.CheckAuth(uinfo, ip); err != nil {
		reason := strings.ToLower(err.Error())
		reason = strings.Replace(reason, " ", "-", -1)
		reason = reason + ".status"
		sc(reason, socks5.StatusConnectionNotAllowedByRuleset)
	}

	if cs.parent.IsExpired() {
		sc("proxies-expired.status", socks5.StatusConnectionNotAllowedByRuleset)
		return
	}

	if !cs.parent.HasBW() {
		debug(99, fmt.Sprintf("[Socks](%s)", cs.parent.Addr), cs.parent.Addr, "Low Bandwidth")
		sc("low-bandwidth.status", socks5.StatusConnectionNotAllowedByRuleset)
		return
	}

	if cs.parent.limiter.Add() == false {

		sc("max-threads-reached.status", socks5.StatusConnectionNotAllowedByRuleset)
		return
	}
	defer cs.parent.limiter.Done()

	proxy, err := cs.parent.SelectProxy(uinfo, m)

	if proxy == nil || err != nil {
		sc("proxy-offline.status", socks5.StatusNetworkUnreachable)
		debugf(99, "[Socks](%s) proxy-offline: %s", cs.parent.Addr, err)
		return
	}

	tunnel, err := proxy.GetTunnel(upstreamHost, upstreamPort)

	if err != nil || tunnel == nil {
		debugf(99, "[Socks](%s) proxy: [%s]%s, proxy-unreachable: %s", cs.parent.Addr, proxy.Type, proxy.Addr(), err)
		sc("proxy-unreachable.status", socks5.StatusNetworkUnreachable)
		return
	}

	defer cs.parent.RunAccountant("SOCKS5", c, tunnel)

	sc("succeeded.status", socks5.StatusSucceeded)

	cs.parent.RunPiper(c, tunnel)

	return
}
