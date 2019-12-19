// mapping to structs in `runtime` package
package proc

import (
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
