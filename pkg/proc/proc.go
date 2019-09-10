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
	BinPath      string
	ThreadNum    int
	GoroutineNum int
	GoVersion    string
}

func (s PSummary) String() string {
	return fmt.Sprintf("bin: %s, goVer: %s, threads: %d, goroutines: %d",
		s.BinPath, s.GoVersion, s.ThreadNum, s.GoroutineNum)
}

// Process wrap operations on target process
type Process struct {
	ID         int
	bin        *gbin.Binary
	threads    map[int]*Thread
	leadThread *Thread
	memFile    *os.File
	pLock      *sync.Mutex // ensure one ptrace one time.

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

func (p *Process) GetGoroutines() ([]*G, error) {
	if err := p.Attach(); err != nil {
		return nil, err
	}
	defer p.Detach()
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
	// TODO parse goroutines concurrently
	for i := uint64(0); i < allglen; i++ {
		gAddr := allgs + i*POINTER_SIZE
		addr, err := p.ReadVMA(gAddr)
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

func (p *Process) parseG(gaddr uint64) (*G, error) {
	gstruct := p.bin.GStruct
	gmb := gstruct.Members
	buf := make([]byte, gstruct.Size)
	if err := p.ReadData(buf, gaddr); err != nil {
		return nil, err
	}
	addr := uint64(gmb["atomicstatus"].StrtOffset)
	status := toUint32(buf[addr : addr+8])
	if status == gdead {
		return &G{Status: gstatus(status)}, nil
	}
	addr = uint64(gmb["gopc"].StrtOffset)
	goPC := toUint64(buf[addr : addr+8])
	addr = uint64(gmb["startpc"].StrtOffset)
	startPC := toUint64(buf[addr : addr+8])
	addr = uint64(gmb["waitreason"].StrtOffset)
	waitreason := gwaitReason(buf[addr])
	addr = uint64(gmb["goid"].StrtOffset)
	goid := toUint64(buf[addr : addr+8])

	addr = uint64(gmb["m"].StrtOffset)
	maddr := toUint64(buf[addr : addr+8])
	m, err := p.parseM(maddr)
	if err != nil {
		return nil, err
	}
	g := &G{
		ID:         goid,
		Status:     gstatus(status),
		WaitReason: gwaitReason(waitreason),
		M:          m,
		GoLoc:      p.getLocation(goPC),
		StartLoc:   p.getLocation(startPC),
	}
	return g, nil
}

func (p *Process) getLocation(addr uint64) *gbin.Location {
	return p.bin.PCToFunc(addr)
}

func (p *Process) parseM(maddr uint64) (*M, error) {
	if maddr == 0 {
		return nil, nil
	}
	mstruct := p.bin.MStruct
	buf := make([]byte, POINTER_SIZE)
	// m.procid is thread id:
	// https://github.com/golang/go/blob/release-branch.go1.13/src/runtime/os_linux.go#L336
	if err := p.ReadData(buf, maddr+uint64(mstruct.Members["procid"].StrtOffset)); err != nil {
		return nil, err
	}
	return &M{ID: toUint64(buf)}, nil
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
		t := &Thread{ID: tid, proc: p}
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
func (p *Process) Summary() (*PSummary, error) {
	if err := p.Attach(); err != nil {
		return nil, err
	}
	defer p.Detach()

	gs, err := p.leadThread.GetGoroutines()
	if err != nil {
		return nil, err
	}
	goVer, err := p.leadThread.GoVersion()
	if err != nil {
		return nil, err
	}

	sum := &PSummary{BinPath: p.bin.Path, ThreadNum: len(p.threads),
		GoroutineNum: len(gs), GoVersion: goVer}

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
func New(pid int) (*Process, error) {
	// TODO support pass in external debug binary
	bin, err := gbin.Load(pid, "")
	if err != nil {
		return nil, err
	}
	if err := bin.Initialize(); err != nil {
		return nil, err
	}
	memFile, err := os.Open(fmt.Sprintf("/proc/%d/mem", pid))
	if err != nil {
		return nil, err
	}
	p := &Process{
		ID:             pid,
		bin:            bin,
		pLock:          new(sync.Mutex),
		memFile:        memFile,
		threads:        make(map[int]*Thread),
		ptraceChan:     make(chan func()),
		ptraceDoneChan: make(chan interface{})}
	go p.handlePtraceFuncs()
	return p, nil
}
