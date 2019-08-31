package process

import (
	"syscall"

	"fmt"
	"io/ioutil"
	"strconv"
)

// Process wrap operations on target process
type Process struct {
	Pid int
}

func (p *Process) Threads() ([]*Thread, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/task", p.Pid))
	if err != nil {
		return nil, err
	}
	ts := make([]*Thread, 0)
	for i, f := range files {
		tid, err := strconv.Atoi(f.Name())
		if err != nil {
			return nil, err
		}
		if tid != p.Pid {
			ts[i] = &Thread{Tid: tid}
		}
	}
	return ts, nil
}

func New(pid int) *Process {
	return &Process{Pid: pid}
}

// Thread wrap operations on a system thread
type Thread struct {
	Tid int
}

// Registers will return thread register address via syscall PTRACE_GETREGS
func (t *Thread) Registers() (*syscall.PtraceRegs, error) {
	var regs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(t.Tid, &regs)
	if err != nil {
		return nil, err
	}
	return &regs, nil
}
