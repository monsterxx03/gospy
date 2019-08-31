package main

import (
	"debug/elf"
	"fmt"
	"os"
	"strconv"

	"gospy/pkg/procmaps"
	"gospy/pkg/process"
)

func LoadBinary(path string) (*elf.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	_elf, err := elf.NewFile(f)
	if err != nil {
		return nil, err
	}
	return _elf, nil
}

func dumpStack(pid int) {
	maps, err := procmaps.ReadProcMaps(pid)
	if err != nil {
		panic(err)
	}
	fstRng := maps[0]
	_elf, err := LoadBinary(fstRng.Filename)
	if err != nil {
		panic(err)
	}
	var progHdr elf.ProgHeader
	foundHdr := false
	for _, phr := range _elf.Progs {
		// find exectuable PT_LOAD program header, it's base!
		if phr.Type == elf.PT_LOAD && (phr.Flags&elf.PF_X > 0) {
			progHdr = phr.ProgHeader
			foundHdr = true
			break
		}
	}
	if !foundHdr {
		panic("didn't find PT_LOAD header in elf file")
	}
	// elf's start vma (virtual memory address) always be 0x400000,
	// baseAddr is the starting offset, should be 0
	baseAddr := fstRng.Start - progHdr.Vaddr
	fmt.Println(baseAddr)
}

func main() {
	pid, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	p := process.New(pid)
	ts, err := p.Threads()
	if err != nil {
		panic(err)
	}
}
