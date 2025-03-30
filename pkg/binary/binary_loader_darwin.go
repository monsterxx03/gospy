//go:build darwin

package binary

/*
#include <libproc.h>
#include <stdlib.h>
*/
import "C"

import (
	"debug/gosym"
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"sync"
	"unsafe"
)

type DarwinBinaryLoader struct {
	file     *macho.File
	path     string
	goSymtab *gosym.Table

	// Cache control
	loadOnce     sync.Once // Ensures symbols are loaded only once
	symbols      map[string]uint64
	loadErr      error
	pcCache      map[uint64]*FuncLoc
	pcCacheMutex sync.RWMutex

	dwarf *dwarfLoader
}

func NewBinaryLoader() BinaryLoader {
	return &DarwinBinaryLoader{}
}

func (d *DarwinBinaryLoader) LoadByPid(pid int) error {
	var pathBuffer [C.PROC_PIDPATHINFO_MAXSIZE]C.char
	ret, err := C.proc_pidpath(C.int(pid), unsafe.Pointer(&pathBuffer[0]), C.PROC_PIDPATHINFO_MAXSIZE)
	if ret <= 0 {
		return fmt.Errorf("failed to get process path: %v", err)
	}

	exePath := C.GoString(&pathBuffer[0])
	return d.Load(exePath)
}

func (d *DarwinBinaryLoader) GetFile() any {
	return d.file
}

func (d *DarwinBinaryLoader) PtrSize() int {
	if d.file == nil {
		// Default to 64-bit pointer size
		return 8
	}

	// Determine pointer size based on Mach-O CPU type
	if d.file.Cpu == macho.Cpu386 {
		return 4
	}
	return 8
}

func (d *DarwinBinaryLoader) Load(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return ErrBinaryNotFound
	}

	file, err := macho.Open(filePath)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidExecutable, err)
	}

	symtab, err := getGoSymtab(file)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidExecutable, err)
	}
	d.goSymtab = symtab

	d.file = file
	d.path = filePath
	d.dwarf = newDwarfLoader(file)
	d.pcCache = make(map[uint64]*FuncLoc)
	return nil
}

func (d *DarwinBinaryLoader) GetSymbols() (map[string]uint64, error) {
	// This will execute the loading function exactly once
	d.loadOnce.Do(func() {
		if d.file.Symtab == nil {
			d.loadErr = errors.New("no symbol table found in binary")
			return
		}

		d.symbols = make(map[string]uint64)
		for _, sym := range d.file.Symtab.Syms {
			if sym.Name != "" {
				d.symbols[sym.Name] = sym.Value
			}
		}
	})

	return d.symbols, d.loadErr
}

func (d *DarwinBinaryLoader) FindVariableAddress(varName string) (uint64, error) {
	symbols, err := d.GetSymbols()
	if err != nil {
		return 0, fmt.Errorf("failed to get symbols: %w", err)
	}

	addr, exists := symbols[varName]
	if exists {
		return addr, nil
	}

	return 0, ErrSymbolNotFound
}

func (d *DarwinBinaryLoader) PCToFuncLoc(addr uint64) *FuncLoc {
	if d.goSymtab == nil {
		return nil
	}

	// Check cache first
	d.pcCacheMutex.RLock()
	if cached, ok := d.pcCache[addr]; ok {
		d.pcCacheMutex.RUnlock()
		return cached
	}
	d.pcCacheMutex.RUnlock()

	file, line, fn := d.goSymtab.PCToLine(addr)
	if fn == nil {
		return nil
	}

	loc := &FuncLoc{
		PC:   addr,
		File: file,
		Line: line,
		Func: fn,
	}

	// Store in cache
	d.pcCacheMutex.Lock()
	if d.pcCache == nil {
		d.pcCache = make(map[uint64]*FuncLoc)
	}
	d.pcCache[addr] = loc
	d.pcCacheMutex.Unlock()

	return loc
}

func (d *DarwinBinaryLoader) GetDWARFLoader() (DWARFLoader, error) {
	if d.dwarf == nil {
		return nil, errors.New("DWARF not loaded")
	}
	return d.dwarf, nil
}

func getGoSymtab(f *macho.File) (*gosym.Table, error) {
	s := f.Section("__gopclntab")
	if s == nil {
		return nil, errors.New("missing __gopclntab")
	}
	data, err := s.Data()
	if err != nil {
		return nil, err
	}
	ln := gosym.NewLineTable(data, f.Section("__text").Addr)
	symtab, err := gosym.NewTable([]byte{}, ln)
	if err != nil {
		return nil, err
	}
	return symtab, nil
}
