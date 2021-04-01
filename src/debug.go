package main

import "log"

func debug(level int, p ...interface{}) {
	if level >= Co.DebugLevel {
		log.Println(p...)
	}
}
