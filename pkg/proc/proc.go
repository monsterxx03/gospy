package proc

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strconv"
	"syscall"

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
		if tid != p.ID {
			if err := t.Lock(); err != nil {
				return err
			}
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
	p := &Process{ID: pid, bin: bin, threads: make(map[int]*Thread), ptraceChan: make(chan func()), ptraceDoneChan: make(chan interface{})}
	go p.handlePtraceFuncs()
	return p, nil
}

// Thread wrap operations on a system thread
type Thread struct {
	ID   int
	proc *Process
}

// ReadVM will use read this thread's virtual memory at addr
func (t *Thread) ReadVMA(addr uint64) (uint64, error) {
	// ptrace's result is a long
	data := make([]byte, 8)
	var err error
	t.proc.execPtraceFunc(func() { _, err = syscall.PtracePeekData(t.ID, uintptr(addr), data) })
	if err != nil {
		return 0, err
	}
	vma := binary.LittleEndian.Uint64(data)
	return vma, nil
}

func (t *Thread) Lock() error {
	var err error
	t.proc.execPtraceFunc(func() { err = syscall.PtraceAttach(t.ID) })
	if err != nil {
		return err
	}
	var s syscall.WaitStatus
	if _, err := syscall.Wait4(t.ID, &s, syscall.WALL, nil); err != nil {
		return err
	}
	return nil
}

func (t *Thread) Unlock() error {
	var err error
	t.proc.execPtraceFunc(func() { err = syscall.PtraceDetach(t.ID) })
	return err
}

// Registers will return thread register address via syscall PTRACE_GETREGS
func (t *Thread) Registers() (*syscall.PtraceRegs, error) {
	var regs syscall.PtraceRegs
	var err error
	t.proc.execPtraceFunc(func() { err = syscall.PtraceGetRegs(t.ID, &regs) })
	if err != nil {
		return nil, err
	}
	return &regs, nil
}

func (t *Thread) GetGoroutines() error {
	allglenAddr, err := t.proc.bin.GetVarAddr("runtime.allglen")
	if err != nil {
		log.Println("Failed to get runtime.allglen from binary")
		return err
	}
	allglen, err := t.ReadVMA(allglenAddr)
	if err != nil {
		log.Println("Failed to read vma for runtime.allglen")
		return err
	}
	fmt.Println("goroutine numbers:", allglen)

	allgsAddr, err := t.proc.bin.GetVarAddr("runtime.allgs")
	if err != nil {
		log.Println("Failed to get runtime.allgs from binary")
		return err
	}
	allgs, err := t.ReadVMA(allgsAddr)
	if err != nil {
		log.Println("Failed to read vma for runtime.allgs")
		return err
	}
	fmt.Println("allags:", allgs)
	// loop all groutines
	//for i := uint64(0); i < allgs; i++ {
	//	gAddr := allgs + i*uint64(8) // amd64 pointer size is 8
	//}
	return nil
}
