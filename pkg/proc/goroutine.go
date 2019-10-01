package proc

import (
	"fmt"
	gbin "gospy/pkg/binary"
)

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
	case "caller":
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

func (g *G) Syscall() bool {
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

func (g *G) ThreadID() uint64 {
	if g.M == nil {
		return 0
	}
	return g.M.ID
}

// M is runtime.m struct
type M struct {
	ID     uint64
	ProcID uint64
}

// P (processor) is runtime.p struct
type P struct {
	ID          int32
	Status      pstatus
	Schedtick   uint32
	Syscalltick uint32
	M           *M
	Runqsize    int
}

func (p *P) Idle() bool {
	return p.Status == pidle
}

func (p *P) Running() bool {
	return p.Status == prunning
}

func (p *P) Syscall() bool {
	return p.Status == psyscall
}

func (p *P) Gcstop() bool {
	return p.Status == pgcstop
}

func (p *P) Dead() bool {
	return p.Status == pdead
}

// Sched is the global goroutine scheduler
type Sched struct {
	Nmidle     int32 // number of idle m's waiting for work
	Nmspinning uint32
	Nmfreed    uint32 // cumulative number of freed m's
	Npidle     int32  // number of idle p's
	Ngsys      uint32 // number of system goroutines
	Runqsize   int32  // global runnable queue size
}
