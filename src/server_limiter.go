package main

import (
	"sync"

	"github.com/Alex-Eftimie/utils"
)

// ServerLimiter performs connections / thread limiting
type ServerLimiter struct {
	max     *int
	current int
	m       sync.Mutex
}

// Add adds a connection
func (sl *ServerLimiter) Add() bool {
	sl.m.Lock()
	defer sl.m.Unlock()

	// always return true if connections are not counted
	if sl.max == nil {
		return true
	}
	if sl.current >= *sl.max {
		return false
	}

	sl.current++

	return true
}

// Done removes a connection from the pool
func (sl *ServerLimiter) Done() {
	sl.m.Lock()
	defer sl.m.Unlock()
	sl.current--
	if sl.current < 0 {
		sl.current = 0
	}
}

// Current returns the current count
func (sl *ServerLimiter) Current() int {

	sl.m.Lock()
	defer sl.m.Unlock()
	return sl.current
}

// SetMax updates the max
func (sl *ServerLimiter) SetMax(max *int) {
	utils.Debugf(999, "ServerLimiter SetMax: %d", max)
	sl.m.Lock()
	defer sl.m.Unlock()
	sl.max = max
}
