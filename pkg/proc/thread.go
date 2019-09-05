package proc

import (
	"encoding/binary"
	"fmt"
	"syscall"

	"github.com/golang/glog"

	gbin "gospy/pkg/binary"
)

// Thread wrap operations on a system thread
type Thread struct {
	ID   int
	proc *Process
}

// ReadVM will read virtual memory at addr
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

func (t *Thread) bin() *gbin.Binary {
	return t.proc.bin
}

func (t *Thread) ReadData(data []byte, addr uint64) error {
	var err error
	t.proc.execPtraceFunc(func() { _, err = syscall.PtracePeekData(t.ID, uintptr(addr), data) })
	if err != nil {
		return err
	}
	return nil
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

func (t *Thread) GetGoroutines() error {
	bin := t.bin()
	allglen, err := t.ReadVMA(bin.AllglenAddr)
	if err != nil {
		glog.Errorf("Failed to read vma for runtime.allglen at %d", bin.AllglenAddr)
		return err
	}
	allgs, err := t.ReadVMA(bin.AllgsAddr)
	if err != nil {
		glog.Errorf("Failed to read vma for runtime.allgs at %d", bin.AllgsAddr)
		return err
	}
	// loop all groutines addresses
	for i := uint64(0); i < allglen; i++ {
		gAddr := allgs + i*uint64(8) // amd64 pointer size is 8
		addr, err := t.ReadVMA(gAddr)
		if err != nil {
			return err
		}
		g, err := t.parseG(addr)
		if err != nil {
			return err
		}
		fmt.Println(g)

	}
	return nil
}

func (t *Thread) parseG(gaddr uint64) (*G, error) {
	gstruct := t.bin().GStruct

	buf := make([]byte, 8)
	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["startpc"].StrtOffset)); err != nil {
		return nil, err
	}
	startPC := binary.LittleEndian.Uint64(buf)
	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["waitreason"].StrtOffset)); err != nil {
		return nil, err
	}
	waitreason := buf[0]
	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["atomicstatus"].StrtOffset)); err != nil {
		return nil, err
	}
	status := binary.LittleEndian.Uint32(buf)
	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["goid"].StrtOffset)); err != nil {
		return nil, err
	}
	goid := binary.LittleEndian.Uint64(buf)

	g := &G{
		ID:         goid,
		StartPC:    startPC,
		WaitReason: gwaitReason(waitreason),
		Status:     gstatus(status),
	}
	return g, nil
}
