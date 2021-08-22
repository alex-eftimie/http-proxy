package main

import (
	"sync"
	"time"
)

var SessionTimeout time.Duration

type SessionManager struct {
	Sessions map[string]*hp

	sync.Mutex
}

type hp struct {
	hp      *ProxyInfo
	timeout time.Time
}

func init() {
	SessionTimeout = Co.SessionTimeout
}

func (sm *SessionManager) GetSession(key string) *ProxyInfo {
	sm.Lock()
	defer sm.Unlock()

	if v, ok := sm.Sessions[key]; ok {
		v.timeout = time.Now().Add(SessionTimeout)
		return v.hp
	}

	return nil
}

func (sm *SessionManager) SetSession(key string, p *ProxyInfo) {
	sm.Lock()
	defer sm.Unlock()

	h := &hp{
		hp:      p,
		timeout: time.Now().Add(SessionTimeout),
	}
	sm.Sessions[key] = h
	go func() {
		// wait until when is in the past and then invalidate the session
		for h.timeout.After(time.Now()) {
			time.Sleep(10 * time.Second)
		}
		sm.Lock()
		delete(sm.Sessions, key)
		sm.Unlock()
	}()
	// set expiration
}
