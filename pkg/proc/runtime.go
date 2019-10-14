// mapping to structs in `runtime` package
package proc

import (
	gbin "gospy/pkg/binary"
)

type common struct {
	_p       *Process
	_binStrt *gbin.Strt
}

func (c *common) Init(p *Process, binStrt *gbin.Strt) {
	c._p = p
	c._binStrt = binStrt
}

func (c *common) Process() *Process {
	return c._p
}

func (c *common) BinStrt() *gbin.Strt {
	return c._binStrt
}

type GoStructer interface {
	Init(p *Process, binStrt *gbin.Strt)
	Parse(addr uint64) error
	BinStrt() *gbin.Strt
	Process() *Process
}

type Gobuf struct {
	common
	PC uint64 `name:"pc"`
}

func (b *Gobuf) Parse(addr uint64) error {
	return parse(addr, b)
}

// G is runtime.g struct parsed from process memory and binary dwarf
type G struct {
	common
	ID         uint64         `name:"goid"`         // goid
	Status     gstatus        `name:"atomicstatus"` // atomicstatus
	WaitReason gwaitReason    `name:"waitreason"`   // if Status ==Gwaiting
	Sched      Gobuf          `name:"sched" binStrt:"runtime.gobuf"`
	Startpc    uint64         `name:"startpc"`
	Gopc       uint64         `name:"gopc"`
	M          *M             `name:"m" binStrt:"runtime.m"` // hold worker thread info
	CurLoc     *gbin.Location // runtime location
	UserLoc    *gbin.Location // location of user code, a subset of CurLoc
	GoLoc      *gbin.Location // location of `go` statement that spawed this goroutine
	StartLoc   *gbin.Location // location of goroutine start function
}

func (g *G) Parse(addr uint64) error {
	if err := parse(addr, g); err != nil {
		return err
	}
	if g.Status == gdead {
		return nil
	}
	p := g.Process()
	g.CurLoc = p.getLocation(g.Sched.PC)
	g.GoLoc = p.getLocation(g.Gopc)
	g.StartLoc = p.getLocation(g.Startpc)
	return nil
}

func (g *G) ParsePtrSlice(addr uint64) ([]*G, error) {
	res, err := parseSliceAt(g.Process(), addr)
	if err != nil {
		return nil, err
	}
	result := make([]*G, 0, len(res))
	for _, addr := range res {
		_g := new(G)
		_g.Init(g.Process(), g.BinStrt())
		if err := _g.Parse(addr); err != nil {
			return nil, err
		}
		if _g.Dead() {
			continue
		}
		result = append(result, _g)
	}
	return result, nil
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
	common
	ID     uint64 `name:"id"`
	ProcID uint64 `name:"procid"`
}

func (m *M) Parse(addr uint64) error {
	return parse(addr, m)
}

// P (processor) is runtime.p struct
type P struct {
	common
	ID          int32   `name:"id"`
	Status      pstatus `name:"status"`
	Schedtick   uint32  `name:"schedtick"`
	Syscalltick uint32  `name:"syscalltick"`
	M           *M      `name:"m" binStrt:"runtime.m"`
	Runq        []byte  `name:"runq"`
	Runqsize    int
}

func (p *P) Parse(addr uint64) error {
	if err := parse(addr, p); err != nil {
		return err
	}
	runqsize := 0
	process := p.Process()
	for i := 0; i < len(p.Runq); i += POINTER_SIZE {
		gaddr := toUint64(p.Runq[i : i+POINTER_SIZE])
		if gaddr != 0 {
			// should cache g by gaddr during one snapshot
			g := new(G)
			g.Init(process, process.bin.GStruct)
			if err := g.Parse(gaddr); err != nil {
				return err
			}
			if !g.Dead() {
				runqsize++
			}
		}
	}
	p.Runqsize = runqsize
	return nil
}

func (p *P) ParsePtrSlice(addr uint64) ([]*P, error) {
	res, err := parseSliceAt(p.Process(), addr)
	if err != nil {
		return nil, err
	}
	result := make([]*P, 0, len(res))
	for _, addr := range res {
		_p := new(P)
		_p.Init(p.Process(), p.BinStrt())
		if err := _p.Parse(addr); err != nil {
			return nil, err
		}
		result = append(result, _p)
	}
	return result, nil
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
	common
	Nmidle     int32  `name:"nmidle"` // number of idle m's waiting for work
	Nmspinning uint32 `name:"nmspinning"`
	Nmfreed    uint64 `name:"nmfreed"`  // cumulative number of freed m's
	Npidle     int32  `name:"npidle"`   // number of idle p's
	Ngsys      uint32 `name:"ngsys"`    // number of system goroutines
	Runqsize   int32  `name:"runqsize"` // global runnable queue size
}

func (s *Sched) Parse(addr uint64) error {
	return parse(addr, s)
}

// MemStat hold memory usage and gc info (runtime/mstat.go)
type MemStat struct {
	common
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
	LastGC        uint64  `name:"last_gc_unix"`    // last gc (in unix time)
	GCCPUFraction float64 `name:"gc_cpu_fraction"` // fraction of CPU time used by GC
}

func (m *MemStat) Parse(addr uint64) error {
	return parse(addr, m)
}

type MSpan struct {
	common
	Npages     uint64     `name:"npages"`
	Sweepgen   uint32     `name:"sweepgen"`
	AllocCount uint16     `name:"allocCount"`
	State      mspanstate `name:"state"`
}

func (s *MSpan) Parse(addr uint64) error {
	return parse(addr, s)
}

// MHeap hold process heap info (runtime/mheap.go:mheap)
type MHeap struct {
	common
	Sweepgen   uint32   `name:"sweepgen"` // used to compare with mspan.sweepgen
	MSpans     []*MSpan `name:"allspans" binStrt:"runtime.mspan"`
	PagesInUse uint64   `name:"pagesInUse"` // pages of spans in stats mSpanInUse
	PagesSwept uint64   `name:"pagesSwept"` // pages swept this cycle
}

func (h *MHeap) Parse(addr uint64) error {
	return parse(addr, h)
}