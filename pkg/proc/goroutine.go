package proc

import (
	"fmt"
	"reflect"

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

type GoStructer interface {
	Parse(binStrt *gbin.Strt, data []byte) error
}

// parse will fill fields on `obj` from `data`, `binStrt` holds mapping info
func parse(obj GoStructer, binStrt *gbin.Strt, data []byte) error {
	members := binStrt.Members
	t := reflect.TypeOf(obj).Elem()
	v := reflect.ValueOf(obj).Elem()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := field.Tag.Get("name")
		if name != "" {
			strtField := members[name]
			addr := uint64(strtField.StrtOffset)
			size := uint64(strtField.Size)
			// fill obj's fields
			switch field.Type.Kind() {
			case reflect.Uint64:
				f := toUint64(data[addr : addr+size])
				v.Field(i).SetUint(f)
			case reflect.Uint32:
				f := toUint32(data[addr : addr+size])
				v.Field(i).SetUint(uint64(f))
			case reflect.Uint8:
				f := uint8(data[addr])
				v.Field(i).SetUint(uint64(f))
			case reflect.Int32:
				f := toInt32(data[addr : addr+size])
				v.Field(i).SetInt(int64(f))
			case reflect.Float64:
				f := toFloat64(data[addr : addr+size])
				v.Field(i).SetFloat(f)
			case reflect.Slice:
				v.Field(i).SetBytes(data[addr : addr+size])
			default:
				return fmt.Errorf("unknown type:%+v", field)
			}
		}
	}

	return nil
}

// G is runtime.g struct parsed from process memory and binary dwarf
type G struct {
	ID         uint64         `name:"goid"`         // goid
	Status     gstatus        `name:"atomicstatus"` // atomicstatus
	WaitReason gwaitReason    `name:"waitreason"`   // if Status ==Gwaiting
	Startpc    uint64         `name:"startpc"`
	Gopc       uint64         `name:"gopc"`
	MPtr       uint64         `name:"m"`
	M          *M             // hold worker thread info
	CurLoc     *gbin.Location // runtime location
	UserLoc    *gbin.Location // location of user code, a subset of CurLoc
	GoLoc      *gbin.Location // location of `go` statement that spawed this goroutine
	StartLoc   *gbin.Location // location of goroutine start function
}

func (g *G) Parse(binStrt *gbin.Strt, data []byte) error {
	return parse(g, binStrt, data)
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
	ID          int32   `name:"id"`
	Status      pstatus `name:"status"`
	Schedtick   uint32  `name:"schedtick"`
	Syscalltick uint32  `name:"syscalltick"`
	M           *M
	Runq        []byte `name:"runq"` // must be public to by parsed in reflect
	Runqsize    int
}

func (p *P) Parse(binStrt *gbin.Strt, data []byte) error {
	return parse(p, binStrt, data)
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
	Nmidle     int32  `name:"nmidle"` // number of idle m's waiting for work
	Nmspinning uint32 `name:"nmspinning"`
	Nmfreed    uint64 `name:"nmfreed"`  // cumulative number of freed m's
	Npidle     int32  `name:"npidle"`   // number of idle p's
	Ngsys      uint32 `name:"ngsys"`    // number of system goroutines
	Runqsize   int32  `name:"runqsize"` // global runnable queue size
}

func (s *Sched) Parse(binStrt *gbin.Strt, data []byte) error {
	return parse(s, binStrt, data)
}

// MemStat hold memory usage and gc info (runtime/mstat.go)
type MemStat struct {
	HeapInuse   uint64 `name:"heap_inuse"`   // bytes allocated and not yet freed
	HeapObjects uint64 `name:"heap_objects"` // total number of allocated objects
	HeapSys     uint64 `name:"heap_sys"`     // virtual address space obtained from system for GC'd heap
	HeapLive    uint64 `name:"heap_live"`    // HeapAlloc - (objects not sweeped)

	Nmalloc uint64 `name:"nmalloc"` // number of mallocs
	Nfree   uint64 `name:"nfree"`   // number of frees

	// gc related
	PauseTotalNs  uint64  `name:"pause_total_ns"`
	NumGC         uint32  `name:"numgc"`
	NumForcedGC   uint32  `name:"numforcedgc"`     // number of user-forced GCs
	GCCPUFraction float64 `name:"gc_cpu_fraction"` // fraction of CPU time used by GC
}

func (m *MemStat) Parse(binStrt *gbin.Strt, data []byte) error {
	return parse(m, binStrt, data)
}
