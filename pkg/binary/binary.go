package binary

import (
	"debug/dwarf"
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
	ln := gosym.NewLineTable(lndata, b.bin.Section(".text").Addr)
	symtab, err := gosym.NewTable([]byte{}, ln)
	if err != nil {
		return err
	}
	fmt.Println("xxx", symtab.PCToFunc(addr).Sym.Name)
	data, _ := b.bin.DWARF()
	reader := data.Reader()
	for {
		entry, _ := reader.Next()
		if entry == nil {
			break
		}
		for _, f := range entry.Field {
			switch f.Val.(type) {
			case string:
				if f.Val.(string) == "runtime.allgs" {
					instructions, ok := entry.Val(dwarf.AttrLocation).([]byte)
					if !ok {
						panic(ok)
					}
				}
			}
		}
	}
	return nil
}
