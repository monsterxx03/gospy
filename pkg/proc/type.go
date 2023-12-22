package proc

// from runtime/typekind.go
const (
	kindBool = 1 + iota
	kindInt
	kindInt8
	kindInt16
	kindInt32
	kindInt64
	kindUint
	kindUint8
	kindUint16
	kindUint32
	kindUint64
	kindUintptr
	kindFloat32
	kindFloat64
	kindComplex64
	kindComplex128
	kindArray
	kindChan
	kindFunc
	kindInterface
	kindMap
	kindPtr
	kindSlice
	kindString
	kindStruct
	kindUnsafePointer
	kindMask = (1 << 5) - 1
)

// internal/abi.Type
type Type struct {
	common
	Kind uint8 `name:"Kind_"`
	Str  int32 `name:"Str"`
}

func (c *Type) Parse(addr uint64) error {
	return parse(addr, c)
}

func (c *Type) String() string {
	switch c.Kind & kindMask {
	case kindBool:
		return "bool"
	case kindInt:
		return "int"
	case kindInt8:
		return "int8"
	case kindInt16:
		return "int16"
	case kindInt32:
		return "int32"
	case kindInt64:
		return "int64"
	case kindUint:
		return "uint"
	case kindUint8:
		return "uint8"
	case kindUint16:
		return "uint16"
	case kindUint32:
		return "uint32"
	case kindUint64:
		return "uint64"
	case kindUintptr:
		return "uintptr"
	case kindFloat32:
		return "float32"
	case kindFloat64:
		return "float64"
	case kindComplex64:
		return "complex64"
	case kindComplex128:
		return "complex128"
	case kindArray:
		return "array"
	case kindChan:
		return "chan"
	case kindFunc:
		return "func"
	case kindInterface:
		return "interface"
	case kindMap:
		return "map"
	case kindPtr:
		return "ptr"
	case kindSlice:
		return "slice"
	case kindString:
		return "string"
	case kindStruct:
		return "struct"
	case kindUnsafePointer:
		return "unsafe.Pointer"
	default:
		return "unknown"
	}
}
