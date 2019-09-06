package proc

import (
	"fmt"
)

type gstatus uint32

func (s gstatus) String() string {
	if s < 0 || s >= gstatus(len(gstatusStrings)) {
		return fmt.Sprintf("unknown goroutine status %d", s)
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

func (g *G) Waiting() bool {
	// waiting means this goroutine is blocked in runtime.
	return g.Status == gwaiting
}

func (g *G) Dead() bool {
	// dead means this goroutine is not executing user code.
	// Maybe exited, or just being initialized.
	return g.Status == gdead
}

func (g *G) String() string {
	result := fmt.Sprintf("G%d status: %s ", g.ID, g.Status)
	if g.Status == gwaiting {
		result += fmt.Sprintf("reason: %s", g.WaitReason)
	}
	return result
}
