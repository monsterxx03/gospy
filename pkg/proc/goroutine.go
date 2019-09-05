package proc

import (
	"fmt"
)

type gstatus uint32

func (s gstatus) String() string {
	if s < 0 || s >= gstatus(len(gstatusStrings)) {
		return fmt.Sprintf("unknown goroutine status %s", s)
	}
	return gstatusStrings[s]
}

type gwaitReason uint8

func (w gwaitReason) String() string {
	if w < 0 || w >= gwaitReason(len(gwaitReasonStrings)) {
		return "unknown wait reason"
	}
	return gwaitReasonStrings[w]
}

// G is goroutine struct parsed from process memory and binary dwarf
type G struct {
	ID         uint64      // goid
	GoPC       uint64      // pc of go statement that created this goroutine
	StartPC    uint64      // pc of goroutine function
	PC         uint64      // sched.pc
	Status     gstatus     // atomicstatus
	WaitReason gwaitReason // if Status ==Gwaiting
}
