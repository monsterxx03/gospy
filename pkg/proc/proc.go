package proc

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"sync"

	"github.com/golang/glog"
	gbin "gospy/pkg/binary"
)

// PSummary holds process summary info
type PSummary struct {
	BinPath         string
	Gs              []*G
	Ps              []*P
	Sched           *Sched
	MemStat         *MemStat
	ThreadsTotal    int
	ThreadsSleeping int
	ThreadsStopped  int
	ThreadsRunning  int
	ThreadsZombie   int
	GTotal          int
	GIdle           int
	GRunning        int
	GSyscall        int
	GWaiting        int
	GoVersion       string
	Gomaxprocs      int
}

func (s PSummary) String() string {
	plines := ""
	for _, p := range s.Ps {
		minfo := "nil"
		if p.M != nil {
			minfo = fmt.Sprintf("M%d", p.M.ID)
		}
		l := fmt.Sprintf("P%d %s, schedtick: %d, syscalltick: %d, curM: %s, runqsize: %d\n", p.ID, p.Status.String(), p.Schedtick, p.Syscalltick, minfo, p.Runqsize)
		plines += l
	}
	// TODO simplify and humanize
	return fmt.Sprintf("bin: %s, goVer: %s, gomaxprocs: %d\n"+
		"Sched: NMidle %d, NMspinning %d, NMfreed %d, NPidle %d, NGsys %d, Runqsize: %d \n"+
		"Heap: HeapInUse %s, HeapSys %s, HeapLive %s, HeapObjects %d, Nmalloc %d, Nfree %d\n"+
		"GC: TotalPauseTime %s, NumGC %d, NumForcedGC %d, GCCpu %f\n"+
		"%s"+
		"Threads: %d total, %d running, %d sleeping, %d stopped, %d zombie\n"+
		"Goroutines: %d total, %d idle, %d running, %d syscall, %d waiting\n",
		s.BinPath, s.GoVersion, s.Gomaxprocs,
		s.Sched.Nmidle, s.Sched.Nmspinning, s.Sched.Nmfreed, s.Sched.Npidle, s.Sched.Ngsys, s.Sched.Runqsize,
		humanateBytes(s.MemStat.HeapInuse), humanateBytes(s.MemStat.HeapSys), humanateBytes(s.MemStat.HeapLive), s.MemStat.HeapObjects, s.MemStat.Nmalloc, s.MemStat.Nfree,
		humanateNS(s.MemStat.PauseTotalNs), s.MemStat.NumGC, s.MemStat.NumForcedGC, s.MemStat.GCCPUFraction,
		plines,
		s.ThreadsTotal, s.ThreadsRunning, s.ThreadsSleeping, s.ThreadsStopped, s.ThreadsZombie,
		s.GTotal, s.GIdle, s.GRunning, s.GSyscall, s.GWaiting,
	)
}

// Process wrap operations on target process
type Process struct {
	ID         int
	bin        *gbin.Binary
	threads    map[int]*Thread
	leadThread *Thread
	memFile    *os.File
	pLock      *sync.Mutex // ensure one ptrace one time.
	goVersion  string
	gomaxprocs int

	// to ensure all ptrace cmd run on same thread
	ptraceChan     chan func()
	ptraceDoneChan chan interface{}
}

// ReadVM will read virtual memory at addr
// TODO handle PIE?
func (p *Process) ReadVMA(addr uint64) (uint64, error) {
	var err error
	// ptrace's result is a long
	data := make([]byte, POINTER_SIZE)
	_, err = p.memFile.ReadAt(data, int64(addr))
	if err != nil {
		return 0, err
	}
	vma := toUint64(data)
	return vma, nil
}

func (p *Process) ReadData(data []byte, addr uint64) error {
	var err error
	_, err = p.memFile.ReadAt(data, int64(addr))
	if err != nil {
		return err
	}
	return nil
}

// GetPs return P's in runtime.allp
func (p *Process) GetPs(lock bool) ([]*P, error) {
	if lock {
		if err := p.Attach(); err != nil {
			return nil, nil
		}
		defer p.Detach()
	}
	bin := p.bin
	plen, err := p.Gomaxprocs()
	if err != nil {
		return nil, err
	}
	allp, err := p.ReadVMA(bin.AllpAddr)
	if err != nil {
		glog.Errorf("Failed to vma for runtime.allg at %d", bin.AllpAddr)
		return nil, err
	}
	result := make([]*P, 0, plen)
	for i := 0; i < plen; i++ {
		paddr := allp + uint64(i)*POINTER_SIZE
		addr, err := p.ReadVMA(paddr)
		if err != nil {
			return nil, err
		}
		_p, err := p.parseP(addr)
		if err != nil {
			glog.Errorf("Failed to parse runtime.p at %d", err)
			return nil, err
		}
		result = append(result, _p)
	}
	return result, nil
}

// GetGs return goroutines
func (p *Process) GetGs(lock bool) ([]*G, error) {
	if lock {
		if err := p.Attach(); err != nil {
			return nil, err
		}
		defer p.Detach()
	}
	bin := p.bin

	allglen, err := p.ReadVMA(bin.AllglenAddr)
	if err != nil {
		glog.Errorf("Failed to read vma for runtime.allglen at %d", bin.AllglenAddr)
		return nil, err
	}
	allgs, err := p.ReadVMA(bin.AllgsAddr)
	if err != nil {
		glog.Errorf("Failed to read vma for runtime.allgs at %d", bin.AllgsAddr)
		return nil, err
	}
	// loop all groutines addresses
	result := make([]*G, 0, allglen)
	for i := uint64(0); i < allglen; i++ {
		gaddr := allgs + i*POINTER_SIZE
		addr, err := p.ReadVMA(gaddr)
		if err != nil {
			return nil, err
		}
		g, err := p.parseG(addr)
		if err != nil {
			return nil, err
		}
		if g.Dead() {
			continue
		}

		result = append(result, g)
	}
	// sort.Slice(result, func(i, j int) bool {
	// 	return result[i].ID < result[j].ID
	// })
	return result, nil
}

func (p *Process) parseM(maddr uint64) (*M, error) {
	if maddr == 0 {
		return nil, nil
	}
	strt := p.bin.MStruct
	buf := make([]byte, POINTER_SIZE)
	if err := p.ReadData(buf, strt.GetFieldAddr(maddr, "id")); err != nil {
		return nil, err
	}
	id := toUint64(buf)

	// m.procid is thread id:
	// https://github.com/golang/go/blob/release-branch.go1.13/src/runtime/os_linux.go#L336
	if err := p.ReadData(buf, strt.GetFieldAddr(maddr, "procid")); err != nil {
		return nil, err
	}
	procid := toUint64(buf)
	return &M{ID: id, ProcID: procid}, nil
}

func (p *Process) parseP(paddr uint64) (*P, error) {
	_p := new(P)
	if err := p.parseStruct(paddr, p.bin.PStruct, _p); err != nil {
		return nil, err
	}
	strt := p.bin.PStruct

	// parse P's local queue size
	runqsize := 0
	for i := 0; i < len(_p.Runq); i += POINTER_SIZE {
		gaddr := toUint64(_p.Runq[i : i+POINTER_SIZE])
		if gaddr != 0 {
			// should cache g by gaddr during one snapshot
			g, err := p.parseG(gaddr)
			if err != nil {
				return nil, err
			}
			if !g.Dead() {
				runqsize++
			}
		}
	}
	_p.Runqsize = runqsize

	// parse P's  binding M
	maddr, err := p.ReadVMA(strt.GetFieldAddr(paddr, "m"))
	if err != nil {
		return nil, err
	}
	m, err := p.parseM(maddr)
	if err != nil {
		return nil, err
	}
	_p.M = m
	return _p, nil
}

func (p *Process) parseG(gaddr uint64) (*G, error) {
	// TODO cache during same snapshot
	g := new(G)
	if err := p.parseStruct(gaddr, p.bin.GStruct, g); err != nil {
		return nil, err
	}
	if g.Status == gdead {
		return &G{Status: gdead}, nil
	}

	// parse pc from g.sched
	gobuf := p.bin.GobufStruct
	schedStart := p.bin.GStruct.Members["sched"].StrtOffset
	addr := gaddr + uint64(schedStart+gobuf.Members["pc"].StrtOffset)
	pc, err := p.ReadVMA(addr)
	if err != nil {
		return nil, err
	}
	g.CurLoc = p.getLocation(pc)

	m, err := p.parseM(g.MPtr)
	if err != nil {
		return nil, err
	}
	g.M = m

	// g.CurLoc = g.Sche
	g.GoLoc = p.getLocation(g.Gopc)
	g.StartLoc = p.getLocation(g.Startpc)
	return g, nil
}

func (p *Process) Gomaxprocs() (int, error) {
	if p.gomaxprocs != 0 {
		return p.gomaxprocs, nil
	}
	data := make([]byte, 4)
	err := p.ReadData(data, p.bin.GomaxprocsAddr)
	if err != nil {
		return 0, err
	}
	p.gomaxprocs = int(toUint32(data))
	return p.gomaxprocs, nil
}

func (p *Process) GoVersion() (string, error) {
	if p.goVersion != "" {
		return p.goVersion, nil
	}
	// it's possible to parse it from binary, not runtime.
	// but I don't know how to do it yet...
	str, err := p.parseString(p.bin.GoVerAddr)
	if err != nil {
		return "", err
	}
	if len(str) <= 2 || str[:2] != "go" {
		return "", fmt.Errorf("invalid go version: %s", str)
	}
	p.goVersion = str[2:]
	return p.goVersion, nil
}

func (p *Process) SchedInfo() (*Sched, error) {
	sched := new(Sched)
	if err := p.parseStruct(p.bin.SchedAddr, p.bin.SchedtStruct, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

func (p *Process) MemStat() (*MemStat, error) {
	mem := new(MemStat)
	if err := p.parseStruct(p.bin.MStatsAddr, p.bin.MStatsStruct, mem); err != nil {
		return nil, err
	}
	return mem, nil
}

func (p *Process) parseString(addr uint64) (string, error) {
	bin := p.bin

	// go string is dataPtr(8 bytes) + len(8 bytes), we can parse string
	// struct from binary with t.bin().Parse, but since its
	// structure is fixed, we can parse directly here.
	dataPtr, err := p.ReadVMA(bin.GoVerAddr)
	if err != nil {
		return "", err
	}
	strLen, err := p.ReadVMA(bin.GoVerAddr + POINTER_SIZE)
	if err != nil {
		return "", err
	}
	blocks := make([]byte, strLen)
	if err := p.ReadData(blocks, dataPtr); err != nil {
		return "", err
	}
	return string(blocks), nil
}

func (p *Process) parseStruct(addr uint64, binStrt *gbin.Strt, strter GoStructer) error {
	buf := make([]byte, binStrt.Size)
	if err := p.ReadData(buf, addr); err != nil {
		return err
	}
	if err := strter.Parse(binStrt, buf); err != nil {
		return err
	}
	return nil
}

func (p *Process) getLocation(addr uint64) *gbin.Location {
	return p.bin.PCToFunc(addr)
}

// Attach will attach to all threads
func (p *Process) Attach() error {
	p.pLock.Lock()
	files, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/task", p.ID))
	if err != nil {
		return err
	}
	for _, f := range files {
		tid, err := strconv.Atoi(f.Name())
		if err != nil {
			return err
		}
		t, err := NewThread(tid, p)
		if err != nil {
			return err
		}
		p.threads[tid] = t
		if err := t.Attach(); err != nil {
			return err
		}
		if tid == p.ID {
			p.leadThread = t
		}
	}
	return nil
}

func (p *Process) Detach() error {
	for _, t := range p.threads {
		if err := t.Detach(); err != nil {
			return err
		}
	}
	p.pLock.Unlock()
	return nil
}

// Summary process info
func (p *Process) Summary(lock bool) (*PSummary, error) {
	if lock {
		if err := p.Attach(); err != nil {
			return nil, err
		}
		defer p.Detach()
	}

	goVer, err := p.GoVersion()
	if err != nil {
		return nil, err
	}
	gomaxprocs, err := p.Gomaxprocs()
	if err != nil {
		return nil, err
	}
	sched, err := p.SchedInfo()
	if err != nil {
		return nil, err
	}
	// threads
	trunning, tsleeping, tstopped, tzombie := 0, 0, 0, 0
	for _, t := range p.threads {
		if t.Running() {
			trunning++
		} else if t.Sleeping() {
			tsleeping++
		} else if t.Stopped() {
			tstopped++
		} else if t.Zombie() {
			tzombie++
		}
	}
	// gs
	gs, err := p.GetGs(false)
	if err != nil {
		return nil, err
	}
	gidle, grunning, gsyscall, gwaiting := 0, 0, 0, 0
	for _, g := range gs {
		if g.Idle() {
			gidle++
		} else if g.Running() {
			grunning++
		} else if g.Syscall() {
			gsyscall++
		} else if g.Waiting() {
			gwaiting++
		}
	}
	// ps
	ps, err := p.GetPs(false)
	if err != nil {
		return nil, err
	}

	memstat, err := p.MemStat()
	if err != nil {
		return nil, err
	}

	sum := &PSummary{BinPath: p.bin.Path, Gs: gs, Ps: ps, ThreadsTotal: len(p.threads), Sched: sched, MemStat: memstat,
		ThreadsRunning: trunning, ThreadsSleeping: tsleeping, ThreadsStopped: tstopped, ThreadsZombie: tzombie,
		GTotal: len(gs), GIdle: gidle, GRunning: grunning, GSyscall: gsyscall, GWaiting: gwaiting,
		GoVersion: goVer, Gomaxprocs: gomaxprocs}

	return sum, nil
}

// GetThread will return target thread on id
func (p *Process) GetThread(id int) (t *Thread, ok bool) {
	t, ok = p.threads[id]
	return
}

// borrowed from delve/proc/native/proc.go
func (p *Process) execPtraceFunc(fn func()) {
	p.ptraceChan <- fn
	<-p.ptraceDoneChan
}

// borrowed from delve/proc/native/proc.go
func (p *Process) handlePtraceFuncs() {
	// We must ensure here that we are running on the same thread during
	// while invoking the ptrace(2) syscall. This is due to the fact that ptrace(2) expects
	// all commands after PTRACE_ATTACH to come from the same thread.
	runtime.LockOSThread()

	for fn := range p.ptraceChan {
		fn()
		p.ptraceDoneChan <- nil
	}
}

// New a Process struct for target pid
func New(pid int, bin string) (*Process, error) {
	var err error
	path := fmt.Sprintf("/proc/%d/exe", pid)
	memFilePath := fmt.Sprintf("/proc/%d/mem", pid)
	if bin != "" {
		path = bin
	}

	b, err := gbin.Load(path)
	if err != nil {
		return nil, err
	}
	if err := b.Initialize(); err != nil {
		return nil, err
	}
	memFile, err := os.Open(memFilePath)
	if err != nil {
		return nil, err
	}
	p := &Process{
		ID:             pid,
		bin:            b,
		pLock:          new(sync.Mutex),
		memFile:        memFile,
		threads:        make(map[int]*Thread),
		ptraceChan:     make(chan func()),
		ptraceDoneChan: make(chan interface{})}
	go p.handlePtraceFuncs()
	return p, nil
}
