package proc

import (
	"encoding/binary"
	"fmt"
	"sort"
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
	data := make([]byte, POINTER_SIZE)
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

func (t *Thread) GetGoroutines() ([]*G, error) {
	bin := t.bin()
	allglen, err := t.ReadVMA(bin.AllglenAddr)
	if err != nil {
		glog.Errorf("Failed to read vma for runtime.allglen at %d", bin.AllglenAddr)
		return nil, err
	}
	allgs, err := t.ReadVMA(bin.AllgsAddr)
	if err != nil {
		glog.Errorf("Failed to read vma for runtime.allgs at %d", bin.AllgsAddr)
		return nil, err
	}
	// loop all groutines addresses
	result := make([]*G, 0)
	// TODO parse goroutines concurrently
	for i := uint64(0); i < allglen; i++ {
		gAddr := allgs + i*POINTER_SIZE
		addr, err := t.ReadVMA(gAddr)
		if err != nil {
			return nil, err
		}
		g, err := t.parseG(addr)
		if err != nil {
			return nil, err
		}
		if g.Dead() {
			continue
		}
		result = append(result, g)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (t *Thread) getLocation(addr uint64) *Location {
	file, ln, fn := t.bin().PCToFunc(addr)
	if fn == nil {
		return nil
	}
	loc := &Location{PC: addr, File: file, Line: ln, Func: fn}
	return loc
}

func (t *Thread) GoVersion() (string, error) {
	// it's possible to parse it from binary, not runtime.
	// but I don't know how to do it yet...
	str, err := t.parseString(t.bin().GoVerAddr)
	if err != nil {
		return "", err
	}
	if len(str) <= 2 || str[:2] != "go" {
		return "", fmt.Errorf("invalid go version: %s", str)
	}
	return str[2:], nil
}

func (t *Thread) parseG(gaddr uint64) (*G, error) {
	gstruct := t.bin().GStruct

	// TODO use process_vm_readv to bulk read goroutine memory data
	buf := make([]byte, POINTER_SIZE)
	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["gopc"].StrtOffset)); err != nil {
		return nil, err
	}
	goPC := toUint64(buf)
	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["startpc"].StrtOffset)); err != nil {
		return nil, err
	}
	startPC := toUint64(buf)
	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["waitreason"].StrtOffset)); err != nil {
		return nil, err
	}
	waitreason := buf[0]
	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["atomicstatus"].StrtOffset)); err != nil {
		return nil, err
	}
	status := toUint32(buf)
	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["goid"].StrtOffset)); err != nil {
		return nil, err
	}
	goid := toUint64(buf)

	if err := t.ReadData(buf, gaddr+uint64(gstruct.Members["m"].StrtOffset)); err != nil {
		return nil, err
	}
	maddr := toUint64(buf)
	m, err := t.parseM(maddr)
	if err != nil {
		return nil, err
	}
	g := &G{
		ID:         goid,
		Status:     gstatus(status),
		WaitReason: gwaitReason(waitreason),
		M:          m,
		GoLoc:      t.getLocation(goPC),
		StartLoc:   t.getLocation(startPC),
	}
	return g, nil
}

func (t *Thread) parseM(maddr uint64) (*M, error) {
	if maddr == 0 {
		return nil, nil
	}
	mstruct := t.bin().MStruct
	buf := make([]byte, POINTER_SIZE)
	// m.procid is thread id:
	// https://github.com/golang/go/blob/release-branch.go1.13/src/runtime/os_linux.go#L336
	if err := t.ReadData(buf, maddr+uint64(mstruct.Members["procid"].StrtOffset)); err != nil {
		return nil, err
	}
	return &M{ID: binary.LittleEndian.Uint64(buf)}, nil
}

func (t *Thread) parseString(addr uint64) (string, error) {
	bin := t.bin()

	// TODO use process_vm_readv can bulk read bytes.
	// go string is dataPtr(8 bytes) + len(8 bytes), we can parse string
	// struct from binary with t.bin().GetStruct("string"), but since its
	// structure is fixed, we can parse directly here.
	dataPtr, err := t.ReadVMA(bin.GoVerAddr)
	if err != nil {
		return "", err
	}
	strLen, err := t.ReadVMA(bin.GoVerAddr + POINTER_SIZE)
	if err != nil {
		return "", err
	}
	blocks := make([]byte, 0, strLen)
	for i := uint64(0); i < strLen; i = i + POINTER_SIZE {
		buf := make([]byte, POINTER_SIZE)
		err := t.ReadData(buf, dataPtr+i*POINTER_SIZE)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, buf...)
	}
	return string(blocks[:strLen]), nil
}
