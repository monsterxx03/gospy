package proc

import (
	"encoding/binary"
	"fmt"
	"log"
	"syscall"
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
	if err := t.proc.bin.GetStruct("runtime.g"); err != nil {
		return err
	}
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
	for i := uint64(0); i < allglen; i++ {
		gAddr := allgs + i*uint64(8) // amd64 pointer size is 8
		// TODO read size runtime.g
		fmt.Println(gAddr)
	}
	return nil
}
