package proc

import (
	"fmt"
	"io/ioutil"
	"strings"
	"syscall"

	gbin "gospy/pkg/binary"
)

// Thread wrap operations on a system thread
type Thread struct {
	ID    int
	proc  *Process
	state string
}

func (t *Thread) bin() *gbin.Binary {
	return t.proc.bin
}

func (t *Thread) State() string {
	return threadStateStrings[t.state]
}

func (t *Thread) Running() bool {
	return t.state == "R"
}

func (t *Thread) Sleeping() bool {
	return t.state == "S"
}

func (t *Thread) Stopped() bool {
	return t.state == "T"
}

func (t *Thread) Zombie() bool {
	return t.state == "Z"
}

func (t *Thread) Attach() error {
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

func (t *Thread) Detach() error {
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

func NewThread(tid int, proc *Process) (*Thread, error) {
	statPath := fmt.Sprintf("/proc/%d/task/%d/stat", proc.ID, tid)
	b, err := ioutil.ReadFile(statPath)
	if err != nil {
		return nil, err
	}
	state := strings.Split(string(b), " ")[2]

	t := &Thread{ID: tid, proc: proc, state: state}
	return t, nil
}
