package proc

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"

	gbin "gospy/pkg/binary"
)

// PSummary holds process summary info
type PSummary struct {
	BinPath string
	Tnum    int // thread number
	Gnum    int // goroutine number
}

// Process wrap operations on target process
type Process struct {
	ID         int
	bin        *gbin.Binary
	threads    map[int]*Thread
	leadThread *Thread

	// to ensure all ptrace cmd run on same thread
	ptraceChan     chan func()
	ptraceDoneChan chan interface{}
}

// Attach will attach to all threads
func (p *Process) Attach() error {
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
	return nil
}

// Summary process info
func (p *Process) Summary() (*PSummary, error) {
	if err := p.Attach(); err != nil {
		return nil, err
	}
	defer p.Detach()

	return nil, nil
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
	p := &Process{
		ID:             pid,
		bin:            bin,
		threads:        make(map[int]*Thread),
		ptraceChan:     make(chan func()),
		ptraceDoneChan: make(chan interface{})}
	go p.handlePtraceFuncs()
	return p, nil
}
