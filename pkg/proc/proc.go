package proc

import (
	"syscall"

	"fmt"
	"io/ioutil"
	"strconv"
)

// Process wrap operations on target process
type Process struct {
	ID int
}

func (p *Process) Threads() ([]*Thread, error) {
	files, err := ioutil.ReadDir(fmt.Sprintf("/proc/%d/task", p.ID))
	if err != nil {
		return nil, err
	}
	ts := make([]*Thread, 0)
	for _, f := range files {
		tid, err := strconv.Atoi(f.Name())
		if err != nil {
			return nil, err
		}
		if tid != p.ID {
			ts = append(ts, &Thread{ID: tid})
		}
	}
	return ts, nil
}

func New(pid int) *Process {
	return &Process{ID: pid}
}

// Thread wrap operations on a system thread
type Thread struct {
	ID int
}

func (t *Thread) Lock() error {
	// Be carefull, must call syscall.PtraceDetach later, otherwise thread will be `zombie`
	if err := syscall.PtraceAttach(t.ID); err != nil {
		return err
	}
	var s syscall.WaitStatus
	if _, err := syscall.Wait4(t.ID, &s, syscall.WALL, nil); err != nil {
		return err
	}
	return nil
}

func (t *Thread) Unlock() error {
	if err := syscall.PtraceDetach(t.ID); err != nil {
		return err
	}
	return nil
}

// Registers will return thread register address via syscall PTRACE_GETREGS
func (t *Thread) Registers() (*syscall.PtraceRegs, error) {
	var regs syscall.PtraceRegs
	err := syscall.PtraceGetRegs(t.ID, &regs)
	if err != nil {
		return nil, err
	}
	return &regs, nil
}
