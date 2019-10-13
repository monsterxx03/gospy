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

type mspanstate uint8

func (s mspanstate) String() string {
	if s < 0 || s >= mspanstate(len(mspanStateStrings)) {
		return fmt.Sprintf("unknown mspan state %d", s)
	}
	return mspanStateStrings[s]
}

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

// parse will fill fields in `obj` by reading memory start from `baseAddr`
// XX: reflect is disguesting, but ...
func parse(baseAddr uint64, obj GoStructer) error {
	p := obj.Process()
	binStrt := obj.BinStrt()
	data := make([]byte, binStrt.Size)
	if err := p.ReadData(data, baseAddr); err != nil {
		return err
	}
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
			case reflect.Ptr:
				_addr := toUint64(data[addr : addr+size])
				if _addr == 0 {
					continue
				}
				bname := field.Tag.Get("binStrt")
				if bname == "" {
					return fmt.Errorf("pointer field %+v don't have `binStrt` tag", field)
				}
				bstrt, ok := p.bin.StrtMap[bname]
				if !ok {
					return fmt.Errorf("can't find %s in p.bin", bname)
				}
				strt := reflect.New(v.Field(i).Type().Elem())
				// call Init dynamically
				strt.MethodByName("Init").Call([]reflect.Value{reflect.ValueOf(p), reflect.ValueOf(bstrt)})
				// recursive parse to fillin  instance
				if err := parse(_addr, strt.Interface().(GoStructer)); err != nil {
					return err
				}
				v.Field(i).Set(strt)
			case reflect.Uint64:
				f := toUint64(data[addr : addr+size])
				v.Field(i).SetUint(f)
			case reflect.Uint32:
				f := toUint32(data[addr : addr+size])
				v.Field(i).SetUint(uint64(f))
			case reflect.Uint16:
				f := toUint16(data[addr : addr+size])
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
				// check on slice item type
				switch field.Type.Elem().Kind() {
				case reflect.Uint8:
					v.Field(i).SetBytes(data[addr : addr+size])
					continue
				case reflect.Ptr:
					if size != POINTER_SIZE*3 {
						// arrayptr + len + cap
						return fmt.Errorf("Invalid size %d for slice of pointer", size)
					}
					arrayptr := toUint64(data[addr : addr+POINTER_SIZE])
					slen := toUint64(data[addr+POINTER_SIZE : addr+POINTER_SIZE*2])
					scap := toUint64(data[addr+POINTER_SIZE*2 : addr+POINTER_SIZE*3])
					slice := reflect.MakeSlice(reflect.SliceOf(field.Type.Elem()), 0, int(scap))
					bname := field.Tag.Get("binStrt")
					if bname == "" {
						return fmt.Errorf("slice field %+v don't have `binStrt` tag", field)
					}
					bstrt, ok := p.bin.StrtMap[bname]
					if !ok {
						return fmt.Errorf("can't find %s in p.bin", bname)
					}
					sliceData := make([]byte, slen*POINTER_SIZE)
					// bulk read array data on arrayptr
					if err := p.ReadData(sliceData, arrayptr); err != nil {
						return err
					}
					for j := uint64(0); j < slen; j++ {
						// rebuild slice items
						strt := reflect.New(field.Type.Elem().Elem())
						// call Init dynamically
						strt.MethodByName("Init").Call([]reflect.Value{reflect.ValueOf(p), reflect.ValueOf(bstrt)})
						idx := j * POINTER_SIZE
						if err := parse(toUint64(sliceData[idx:idx+POINTER_SIZE]), strt.Interface().(GoStructer)); err != nil {
							return err
						}
						slice = reflect.Append(slice, strt)
					}
					v.Field(i).Set(slice)
					continue
				default:
					return fmt.Errorf("Unsupport slice item %+v", field)
				}
			default:
				return fmt.Errorf("unknown type:%+v", field)
			}
		}
	}

	return nil
}

// G is runtime.g struct parsed from process memory and binary dwarf
type G struct {
	common
	ID         uint64         `name:"goid"`         // goid
	Status     gstatus        `name:"atomicstatus"` // atomicstatus
	WaitReason gwaitReason    `name:"waitreason"`   // if Status ==Gwaiting
	Startpc    uint64         `name:"startpc"`
	Gopc       uint64         `name:"gopc"`
	M          *M             `name:"m" binStrt:"runtime.m"` // hold worker thread info
	CurLoc     *gbin.Location // runtime location
	UserLoc    *gbin.Location // location of user code, a subset of CurLoc
	GoLoc      *gbin.Location // location of `go` statement that spawed this goroutine
	StartLoc   *gbin.Location // location of goroutine start function
}

func (g *G) Parse(addr uint64) error {
	return parse(addr, g)
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
	Runq        []byte  `name:"runq"` // must be public to by parsed in reflect
	Runqsize    int
}

func (p *P) Parse(addr uint64) error {
	return parse(addr, p)
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
