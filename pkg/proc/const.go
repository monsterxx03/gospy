package proc

import (
	"fmt"
)

// amd64 pointer size
const POINTER_SIZE = 8

// from runtime/runtime2.go
const (
	pidle = iota // 0
	prunning
	psyscall
	pgcstop
	pdead
)

var pstatusStrings = [...]string{
	pidle:    "idle",
	prunning: "running",
	psyscall: "syscall",
	pgcstop:  "gcstop",
	pdead:    "dead",
}

// from runtime/runtime2.go
const (
	gidle            = iota // 0
	grunnable               // 1
	grunning                // 2
	gsyscall                // 3
	gwaiting                // 4
	gmoribund_unused        // 5
	gdead                   // 6
	genqueue_unused         // 7
	gcopystack              // 8
	gpreempted              // 9
	gscan            = 0x1000
	gscanrunnable    = gscan + grunnable  // 0x1001
	gscanrunning     = gscan + grunning   // 0x1002
	gscansyscall     = gscan + gsyscall   // 0x1003
	gscanwaiting     = gscan + gwaiting   // 0x1004
	gscanpreempted   = gscan + gpreempted // 0x1009
)

var gstatusStrings = [...]string{
	gidle:            "idle",
	grunnable:        "runnable",
	grunning:         "running",
	gsyscall:         "syscall",
	gwaiting:         "waiting",
	gmoribund_unused: "moribund_unused",
	gdead:            "dead",
	genqueue_unused:  "enqueue_unused",
	gcopystack:       "copystack",
	gpreempted:       "preempted",
	gscan:            "scan",
	gscanrunnable:    "scanrunnable",
	gscanrunning:     "scanrunning",
	gscansyscall:     "scansyscall",
	gscanwaiting:     "scanwaiting",
	gscanpreempted:   "scanpreempted",
}

// from: man proc
var threadStateStrings = map[string]string{
	"R": "Running",
	"S": "Sleeping",
	"D": "Disk sleep",
	"Z": "Zombie",
	"T": "Stopped",
	"t": "Tracing stop",
	"w": "Paging",
	"x": "Dead",
	"X": "Dead",
	"K": "Wakekill",
	"W": "Waking",
	"P": "Parked",
}

const (
	mspanDead = iota
	mspanInUse
	mspanManual
	mspanFree
)

var mspanStateStrings = [...]string{
	mspanDead:   "dead",
	mspanInUse:  "inuse",
	mspanManual: "manual",
	mspanFree:   "free",
}

// stringify status code

type pstatus uint32

func (s pstatus) String() string {
	if s < 0 || s >= pstatus(len(pstatusStrings)) {
		return fmt.Sprintf("unknown processor status %d", s)
	}
	return pstatusStrings[s]
}

type gstatus uint32

func (s gstatus) String() string {
	if s < 0 || s >= gstatus(len(gstatusStrings)) {
		return fmt.Sprintf("unknown goroutine status %d", s)
	}
	return gstatusStrings[s]
}

type mspanstate uint8

func (s mspanstate) String() string {
	if s < 0 || s >= mspanstate(len(mspanStateStrings)) {
		return fmt.Sprintf("unknown mspan state %d", s)
	}
	return mspanStateStrings[s]
}
