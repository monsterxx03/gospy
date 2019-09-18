package proc

import (
	"fmt"
	gbin "gospy/pkg/binary"
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

// G is runtime.g struct parsed from process memory and binary dwarf
type G struct {
	ID         uint64         // goid
	Status     gstatus        // atomicstatus
	WaitReason gwaitReason    // if Status ==Gwaiting
	M          *M             // hold worker thread info
	CurLoc     *gbin.Location // runtime location
	UserLoc    *gbin.Location // location of user code, a subset of CurLoc
	GoLoc      *gbin.Location // location of `go` statement that spawed this goroutine
	StartLoc   *gbin.Location // location of goroutine start function
}

func (g *G) GetLocation(pcType string) *gbin.Location {
	switch pcType {
	case "current":
		return g.CurLoc
	case "go":
		return g.GoLoc
	case "start":
		return g.StartLoc
	default:
		return nil
	}
}

func (g *G) Idle() bool {
	return g.Status == gidle
}

func (g *G) Running() bool {
	return g.Status == grunning
}

func (g *G) Syscalling() bool {
	return g.Status == gsyscall
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
	result := fmt.Sprintf("G%d status: %s, ", g.ID, g.Status)
	if g.Status == gwaiting {
		result += fmt.Sprintf("reason: %s, ", g.WaitReason)
	}
	if g.M != nil {
		result += fmt.Sprintf("thread: %d", g.M.ID)
	}
	return result
}

func (g *G) ThreadID() uint64 {
	if g.M == nil {
		return 0
	}
	return g.M.ID
}

// M is runtime.m struct parsed from process memory and binary dwarf
type M struct {
	ID uint64
}
