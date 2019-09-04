package proc

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"

	gbinary "gospy/pkg/binary"
)

// Process wrap operations on target process
type Process struct {
	ID            int
	bin           *gbinary.Binary
	threads       map[int]*Thread
	currentThread *Thread

	ptraceChan     chan func()
	ptraceDoneChan chan interface{}
}

func (p *Process) UpdateThreads() error {
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
		// TODO maybe neend't attach all threads?
		if err := t.Lock(); err != nil {
			return err
		}
		if p.currentThread == nil {
			p.currentThread = t
		}
	}
	return nil
}

func (p *Process) GetCurrentThread() *Thread {
	return p.currentThread
}

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

func New(pid int) (*Process, error) {
	// TODO support pass in external debug binary
	bin, err := gbinary.Load(pid, "")
	if err != nil {
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
