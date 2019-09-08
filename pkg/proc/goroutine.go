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

type Location struct {
	PC   uint64 // program counter
	File string // source code file name, from dwarf info
	Line int    // soure code line, from dwarf info
	Func string // function name
}

// G is runtime.g struct parsed from process memory and binary dwarf
type G struct {
	ID         uint64      // goid
	Status     gstatus     // atomicstatus
	WaitReason gwaitReason // if Status ==Gwaiting
	M          *M          // hold worker thread info
	CurLoc     *Location   // runtime location
	UserLoc    *Location   // location of user code, a subset of CurLoc
	GoLoc      *Location   // location of `go` statement that spawed this goroutine
	StartLoc   *Location   // location of goroutine start function
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
		result += fmt.Sprintf("reason: %s ", g.WaitReason)
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
