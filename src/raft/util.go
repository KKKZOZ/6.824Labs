package raft

import "log"

// Debugging
const aDebug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if aDebug {
		log.Printf(format, a...)
	}
	return
}
