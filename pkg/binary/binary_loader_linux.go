//go:build linux

package binary

import (
	"debug/elf"
	"debug/gosym"
	"errors"
	"fmt"
	"os"
	"sync"
)

type LinuxBinaryLoader struct {
	file     *elf.File
	path     string
	goSymtab *gosym.Table

	// Cache control
	loadOnce sync.Once // Ensures symbols are loaded only once
	symbols  map[string]uint64
	loadErr  error

	dwarf *dwarfLoader
}

func (l *LinuxBinaryLoader) GetFile() any {
	return l.file
}

func (l *LinuxBinaryLoader) PtrSize() int {
	if l.file.Class == elf.ELFCLASS64 {
		return 8
	}
	return 4
}

func NewBinaryLoader() BinaryLoader {
	return &LinuxBinaryLoader{}
}

func (l *LinuxBinaryLoader) LoadByPid(pid int) error {
	exePath := fmt.Sprintf("/proc/%d/exe", pid)
	targetPath, err := os.Readlink(exePath)
	if err != nil {
		return fmt.Errorf("failed to read process exe link: %w", err)
	}
	return l.Load(targetPath)
}

func (l *LinuxBinaryLoader) Load(filePath string) error {
	file, err := elf.Open(filePath)
	if os.IsNotExist(err) {
		return ErrBinaryNotFound
	}
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidExecutable, err)
	}

	symtab, err := getGoSymtab(file)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidExecutable, err)
	}
	l.goSymtab = symtab

	l.file = file
	l.path = filePath
	l.dwarf = newDwarfLoader(file)
	return nil
}

func (l *LinuxBinaryLoader) GetSymbols() (map[string]uint64, error) {
	// This will execute the loading function exactly once
	l.loadOnce.Do(func() {
		symtab, err := l.file.Symbols()
		if err != nil {
			l.loadErr = fmt.Errorf("failed to get symbols: %w", err)
			return
		}

		l.symbols = make(map[string]uint64)
		for _, sym := range symtab {
			l.symbols[sym.Name] = sym.Value
		}
	})

	return l.symbols, l.loadErr
}

func (l *LinuxBinaryLoader) FindVariableAddress(varName string) (uint64, error) {
	symbols, err := l.GetSymbols()
	if err != nil {
		return 0, err
	}

	addr, exists := symbols[varName]
	if !exists {
		return 0, fmt.Errorf("%w: %q", ErrSymbolNotFound, varName)
	}
	return addr, nil
}

func (l *LinuxBinaryLoader) PCToFuncLoc(addr uint64) *FuncLoc {
	if l.goSymtab == nil {
		return nil
	}

	file, line, fn := l.goSymtab.PCToLine(addr)
	if fn == nil {
		return nil
	}

	return &FuncLoc{
		PC:   addr,
		File: file,
		Line: line,
		Func: fn,
	}
}

func (l *LinuxBinaryLoader) GetDWARFLoader() (DWARFLoader, error) {
	if l.dwarf == nil {
		return nil, errors.New("DWARF not loaded")
	}
	return l.dwarf, nil
}

func getGoSymtab(f *elf.File) (*gosym.Table, error) {
	s := f.Section(".gopclntab")
	if s == nil {
		return nil, errors.New("missing gopclntab")
	}
	data, err := s.Data()
	if err != nil {
		return nil, err
	}
	ln := gosym.NewLineTable(data, f.Section(".text").Addr)
	symtab, err := gosym.NewTable([]byte{}, ln)
	if err != nil {
		return nil, err
	}
	return symtab, nil
}
