// mapping to structs in `runtime` package
package proc

import (
	"fmt"
	gbin "gospy/pkg/binary"
)

type common struct {
	_addr    uint64
	_p       *Process
	_binStrt *gbin.Strt
}

func (c *common) Init(p *Process, binStrt *gbin.Strt, addr uint64) {
	c._addr = addr
	c._p = p
	c._binStrt = binStrt
}

func (c *common) Process() *Process {
	return c._p
}

func (c *common) BinStrt() *gbin.Strt {
	return c._binStrt
}

func (c *common) Addr() uint64 {
	return c._addr
}

type GoStructer interface {
	Init(p *Process, binStrt *gbin.Strt, addr uint64)
	Parse(addr uint64) error
	BinStrt() *gbin.Strt
	Process() *Process
}

type Gobuf struct {
	common
	SP uint64 `name:"sp"`
	PC uint64 `name:"pc"`
}

func (b *Gobuf) Parse(addr uint64) error {
	return parse(addr, b)
}

type stack struct {
	common
	Lo uint64 `name:"lo"`
	Hi uint64 `name:"hi"`
}

func (s *stack) Parse(addr uint64) error {
	return parse(addr, s)
}

// Sudog is runtime.sudog, when a g entering waiting state, it's attached on sudog
type Sudog struct {
	common
	IsSelect    bool   `name:"isSelect"`
	Ticket      uint32 `name:"ticket"`
	AcquireTime int64  `name:"acquiretime"`
	ReleaseTime int64  `name:"releasetime"`
	C           *HChan `name:"c" binStrt:"runtime.hchan"`
}

func (s *Sudog) String() string {
	return fmt.Sprintf("isSelect %t, chan: %+v", s.IsSelect, s.C)
}

func (s *Sudog) Parse(addr uint64) error {
	return parse(addr, s)
}

// HChan is runtime.hchan, result of make(chan xx)
type HChan struct {
	common
	QCount    uint   `name:"qcount"`
	DataqSize uint   `name:"dataqsiz"`
	ElemSize  uint16 `name:"elemsize"`
	ElemType  *Type  `name:"elemtype" binStrt:"runtime._type"`
	Closed    uint32 `name:"closed"`
	Sendx     uint   `name:"sendx"`
	Recvx     uint   `name:"recvx"`
}

func (h *HChan) Parse(addr uint64) error {
	return parse(addr, h)
}

func (h *HChan) String() string {
	return fmt.Sprintf("elemsize: %d, elemtype: %s, dataqsize: %d, addr: %d", h.ElemSize, h.ElemType, h.DataqSize, h.Addr())
}

// G is runtime.g struct parsed from process memory and binary dwarf
type G struct {
	common
	Stack        stack          `name:"stack" binStrt:"runtime.stack"`
	ID           uint64         `name:"goid"`         // goid
	Status       gstatus        `name:"atomicstatus"` // atomicstatus
	WaitReason   gwaitReason    `name:"waitreason"`   // if Status ==Gwaiting
	Sched        Gobuf          `name:"sched" binStrt:"runtime.gobuf"`
	Startpc      uint64         `name:"startpc"`
	Gopc         uint64         `name:"gopc"`
	M            *M             `name:"m" binStrt:"runtime.m"` // hold worker thread info
	WaitingSudog *Sudog         `name:"waiting" binStrt:"runtime.sudog"`
	CurLoc       *gbin.Location // runtime location
	UserLoc      *gbin.Location // location of user code, a subset of CurLoc
	GoLoc        *gbin.Location // location of `go` statement that spawed this goroutine
	StartLoc     *gbin.Location // location of goroutine start function
}

func (g *G) GetWaitReason() (string, error) {
	version, err := g.Process().GoVersion()
	if err != nil {
		return "", err
	}
	return getWaitReasonMap(version)[g.WaitReason], nil
}

func (g *G) StackSize() uint64 {
	return g.Stack.Hi - g.Stack.Lo
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
		_g.Init(g.Process(), g.BinStrt(), addr)
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
