package binary

import (
	"debug/elf"
	"debug/gosym"
	"fmt"
	"os"
)

type Binary struct {
	bin *elf.File
}

func Load(pid int) (*Binary, error) {
	path := fmt.Sprintf("/proc/%d/exe", pid)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	b, err := elf.NewFile(f)
	if err != nil {
		return nil, err
	}
	return &Binary{bin: b}, err
}

func (b *Binary) Search(addr uint64) error {
	lndata, err := b.bin.Section(".gopclntab").Data()
	if err != nil {
		println("wrong wrong line data")
		return err
	}
	symdata, err := b.bin.Section(".gosymtab").Data()
	if err != nil {
		println("wrong symdata")
		return err
	}
	ln := gosym.NewLineTable(lndata, b.bin.Section(".text").Addr)
	symtab, err := gosym.NewTable(symdata, ln)
	if err != nil {
		return err
	}
	fmt.Println("xxx", symtab.PCToFunc(addr).Sym.Name)
	return nil
}
