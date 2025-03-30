package binary

import (
	"debug/gosym"
	"errors"
	"fmt"
)

var (
	ErrBinaryNotFound    = errors.New("binary file not found")
	ErrInvalidExecutable = errors.New("invalid or unsupported executable format")
	ErrSymbolNotFound    = errors.New("symbol not found in binary")
)

// BinaryLoader defines the interface for analyzing Go binaries
type DWARFLoader interface {
	// Check if binary has DWARF info
	HasDWARF() bool
	// Get struct field offset
	GetStructOffset(typeName, fieldName string) (uint64, error)
	// Get nested struct field offset
	GetNestedOffset(outerType, outerField, innerField string) (uint64, error)
	// Get size of a struct type
	GetStructSize(typeName string) (uint64, error)
}

type BinaryLoader interface {
	// Load initializes the binary analysis from a file path
	Load(filePath string) error

	// LoadByPid initializes the binary analysis by process ID
	LoadByPid(pid int) error

	// GetSymbols returns all symbols from the binary
	GetSymbols() (map[string]uint64, error)

	// FindVariableAddress locates a specific variable's memory address
	FindVariableAddress(varName string) (uint64, error)

	// GetFile returns the underlying executable file object
	GetFile() interface{}

	// PtrSize returns the pointer size (4 for 32-bit, 8 for 64-bit)
	PtrSize() int

	PCToFuncLoc(addr uint64) *FuncLoc

	// GetDWARFLoader returns the DWARF loader if available
	GetDWARFLoader() (DWARFLoader, error)
}

type FuncLoc struct {
	PC   uint64      // program counter
	File string      // source code file name, from dwarf info
	Line int         // soure code line, from dwarf info
	Func *gosym.Func // function name
	desc string      // cached description
}

func (f *FuncLoc) Desc() string {
	if f == nil {
		return "<nil>"
	}

	// Return cached description if available
	if f.desc != "" {
		return f.desc
	}

	if f.Func != nil {
		if f.File != "" && f.Line > 0 {
			f.desc = fmt.Sprintf("%s (%s:%d)", f.Func.Name, f.File, f.Line)
			return f.desc
		}
		f.desc = f.Func.Name
		return f.desc
	}

	if f.File != "" && f.Line > 0 {
		f.desc = fmt.Sprintf("%s:%d", f.File, f.Line)
		return f.desc
	}

	f.desc = fmt.Sprintf("0x%x", f.PC)
	return f.desc
}
