package binary

import (
	"debug/dwarf"
	"debug/elf"
	"debug/gosym"
	"encoding/binary"
	"fmt"
	"os"
)

const (
	DW_OP_addr = 0x03

	// type in elf binary
	UTYPE_VAR    string = "Variable"
	UTYPE_STRUCT string = "StructType"
)

type unit struct {
	name  string
	utype string
}

type Location struct {
	PC   uint64      // program counter
	File string      // source code file name, from dwarf info
	Line int         // soure code line, from dwarf info
	Func *gosym.Func // function name
}

func (l Location) String() string {
	//rn := l.Func.ReceiverName()
	//fn := l.Func.Name
	//if rn != "" {
	//	fn = fmt.Sprintf("%s.%s", rn, fn)
	//}
	return fmt.Sprintf("%s (%s:%d)", l.Func.BaseName(), l.File, l.Line)
}

type Binary struct {
	Path      string
	bin       *elf.File
	funcCache map[uint64]*Location
	SymTable  *gosym.Table

	// following fields are parsed from binary dwarf during starting
	GoVerAddr      uint64 // parsed vma of runtime.buildVersion
	GStruct        *Strt  // parsed runtime.g struct
	MStruct        *Strt  // parsed runtime.m struct
	PStruct        *Strt  // parsed runtime.p struct
	GobufStruct    *Strt  // parsed runtime.gobuf struct
	SchedtStruct   *Strt  // parsed runtime.schedt struct
	MStatsStruct   *Strt  // parsed runtime.mstats struct
	SchedAddr      uint64 // parsed vma of runtime.sched
	AllglenAddr    uint64 // parsed vma of runtime.allglen
	AllgsAddr      uint64 // parsed vma of runtime.allgs
	AllpAddr       uint64 // parsed vma of runtime.allp
	GomaxprocsAddr uint64 // parsed vma of runtime.gomaxprocs
	MStatsAddr     uint64 // parsed vma of runtime.memstats
}

// Strt is a abstruct struct parsed from dwarf info
type Strt struct {
	Name    string
	Size    int64
	Members map[string]*StrtMember
}

func (s *Strt) GetFieldAddr(baseAddr uint64, name string) uint64 {
	return baseAddr + uint64(s.Members[name].StrtOffset)
}

type StrtMember struct {
	Name       string
	Size       int64  // member field size
	Offset     uint32 // offset in binary
	StrtOffset int64  // offset inside struct
}

func Load(path string) (*Binary, error) {
	realpath, err := os.Readlink(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	b, err := elf.NewFile(f)
	if err != nil {
		return nil, err
	}
	lnSession := b.Section(".gopclntab")
	if lnSession == nil {
		return nil, fmt.Errorf("Can't find .gopclntab session in binary, not a debug build?")
	}
	lndata, err := lnSession.Data()
	if err != nil {
		return nil, err
	}
	ln := gosym.NewLineTable(lndata, b.Section(".text").Addr)
	// from go 1.3, .gosymtab section is empty, needn't anymore,
	// it's okay to pass an empty byte slice.
	symtab, err := gosym.NewTable([]byte{}, ln)
	if err != nil {
		return nil, err
	}
	return &Binary{Path: realpath, bin: b, funcCache: make(map[uint64]*Location), SymTable: symtab}, nil
}

// Initialize will pre parse some info from elf binary
func (b *Binary) Initialize() error {
	result, err := b.Parse(
		&unit{"runtime.g", UTYPE_STRUCT},
		&unit{"runtime.m", UTYPE_STRUCT},
		&unit{"runtime.p", UTYPE_STRUCT},
		&unit{"runtime.allglen", UTYPE_VAR},
		&unit{"runtime.sched", UTYPE_VAR},
		&unit{"runtime.gobuf", UTYPE_STRUCT},
		&unit{"runtime.schedt", UTYPE_STRUCT},
		&unit{"runtime.allgs", UTYPE_VAR},
		&unit{"runtime.allp", UTYPE_VAR},
		&unit{"runtime.buildVersion", UTYPE_VAR},
		&unit{"runtime.gomaxprocs", UTYPE_VAR},
		&unit{"runtime.mstats", UTYPE_STRUCT},
		&unit{"runtime.memstats", UTYPE_VAR},
	)
	if err != nil {
		return err
	}
	b.GStruct = result["runtime.g"].(*Strt)
	b.MStruct = result["runtime.m"].(*Strt)
	b.PStruct = result["runtime.p"].(*Strt)
	b.AllglenAddr = result["runtime.allglen"].(uint64)
	b.SchedAddr = result["runtime.sched"].(uint64)
	b.GobufStruct = result["runtime.gobuf"].(*Strt)
	b.SchedtStruct = result["runtime.schedt"].(*Strt)
	b.AllgsAddr = result["runtime.allgs"].(uint64)
	b.AllpAddr = result["runtime.allp"].(uint64)
	b.GoVerAddr = result["runtime.buildVersion"].(uint64)
	b.GomaxprocsAddr = result["runtime.gomaxprocs"].(uint64)
	b.MStatsStruct = result["runtime.mstats"].(*Strt)
	b.MStatsAddr = result["runtime.memstats"].(uint64)
	return nil
}

func getEntryName(entry *dwarf.Entry) string {
	for _, f := range entry.Field {
		switch f.Val.(type) {
		case string:
			if f.Attr.String() == "Name" {
				return f.Val.(string)
			}
		}
	}
	return ""
}

func (b *Binary) DumpVar(name string) (Var, error) {
	data, err := b.bin.DWARF()
	if err != nil {
		return nil, err
	}
	reader := data.Reader()
	for {
		entry, err := reader.Next()
		if err != nil {
			return nil, err
		}
		if entry == nil {
			break
		}
		if entry.Tag.String() != UTYPE_VAR {
			continue
		}
		if getEntryName(entry) != name {
			continue
		}
		// parse vma from entry
		addr, err := parseVarAddr(name, entry)
		if err != nil {
			return nil, err
		}
		// parse variable type
		offset := entry.Val(dwarf.AttrType).(dwarf.Offset)
		t, err := data.Type(offset)
		if err != nil {
			return nil, err
		}
		pt, err := parseVarType(name, addr, t)
		if err != nil {
			return nil, err
		}
		return pt, nil
	}
	return nil, fmt.Errorf("Can't find variable %s", name)
}

func parseVarType(name string, addr uint64, t dwarf.Type) (Var, error) {
	switch t.(type) {
	case *dwarf.StructType:
		strt := t.(*dwarf.StructType)
		// go string is a kind of struct
		if strt.StructName == "string" {
			return &StringVar{CommonType{Name: name, Addr: addr, Size: strt.Size()}}, nil
		}
		return nil, fmt.Errorf("unsupported struct type %s", t)
	case *dwarf.UintType:
		u := t.(*dwarf.UintType)
		return &UintVar{CommonType{Name: name, Addr: addr, Size: u.Size()}}, nil
	case *dwarf.IntType:
		i := t.(*dwarf.IntType)
		return &IntVar{CommonType{Name: name, Addr: addr, Size: i.Size()}}, nil
	case *dwarf.BoolType:
		b := t.(*dwarf.BoolType)
		return &BoolVar{CommonType{Name: name, Addr: addr, Size: b.Size()}}, nil
	case *dwarf.PtrType:
		_t := t.(*dwarf.PtrType).Type
		nest_t, err := parseVarType(name, addr, t.(*dwarf.PtrType).Type)
		if err != nil {
			return nil, err
		}
		res := &PtrVar{
			CommonType: CommonType{Name: name, Addr: addr, Size: _t.Size()},
			Type:       nest_t}
		return res, nil
	default:
		return nil, fmt.Errorf("unknown type %s", t)
	}
}

func (b *Binary) Parse(units ...*unit) (map[string]interface{}, error) {
	data, err := b.bin.DWARF()
	if err != nil {
		return nil, err
	}
	umap := make(map[string]*unit)
	for _, u := range units {
		umap[u.name] = u
	}
	result := make(map[string]interface{})
	reader := data.Reader()
	for {
		if len(umap) == 0 {
			// find all targets
			break
		}
		entry, err := reader.Next()
		if err != nil {
			return nil, err
		}
		if entry == nil {
			// reach end
			break
		}
		name := getEntryName(entry)
		if _, ok := umap[name]; !ok {
			continue
		}
		utype := entry.Tag.String()
		switch utype {
		case UTYPE_VAR:
			addr, err := parseVarAddr(name, entry)
			if err != nil {
				return nil, err
			}
			result[name] = addr
			delete(umap, name)
		case UTYPE_STRUCT:
			strt, err := parseStruct(reader, name, entry)
			if err != nil {
				return nil, err
			}
			result[name] = strt
			delete(umap, name)
		default:
			continue
		}
	}
	if len(umap) != 0 {
		return nil, fmt.Errorf("Failed to parse: %+v from binary", umap)
	}
	return result, nil
}

func parseVarAddr(name string, entry *dwarf.Entry) (uint64, error) {
	instructions, ok := entry.Val(dwarf.AttrLocation).([]byte)
	if !ok {
		return 0, fmt.Errorf("Failed to parse variable %v from binary", entry)
	}
	if len(instructions) != 9 {
		return 0, fmt.Errorf("Read invalid DW_AT_location: %v", instructions)
	}
	if instructions[0] != DW_OP_addr {
		return 0, fmt.Errorf("%s's DW_OP type isn't DW_OP_addr(0x03), don't support to parse: %v", name, instructions)
	}
	// parse left 8 bytes as virtual memory address
	addr := uint64(binary.LittleEndian.Uint64(instructions[1:]))
	return addr, nil
}

func parseStruct(reader *dwarf.Reader, name string, entry *dwarf.Entry) (*Strt, error) {
	// example struct entry:
	//	{Offset:240731
	//	 Tag:StructType
	//	 Children:true
	//	 Field:[{Attr:Name Val:runtime.g Class:ClassString}
	//			{Attr:ByteSize Val:376 Class:ClassConstant}
	//			{Attr:Attr(10496) Val:25 Class:ClassConstant}
	//			{Attr:Attr(10500) Val:427680 Class:ClassAddress}]}
	// find next DW_TAG_typedef runtime.g
	// entries between them are member fields
	result := new(Strt)
	result.Name = name
	result.Members = make(map[string]*StrtMember)
	for _, f := range entry.Field {
		if f.Attr.String() == "ByteSize" {
			result.Size = f.Val.(int64)
		}
	}
	err := result.parseMembers(reader)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Strt) parseMembers(reader *dwarf.Reader) error {
	prev := new(StrtMember)
	for {
		entry, err := reader.Next()
		if err != nil {
			return err
		}
		if entry == nil {
			break
		}
		if entry.Tag == 0 {
			// end of struct, calcualte last member's size
			prev.Size = s.Size - int64(prev.StrtOffset)
			break
		}
		if entry.Tag.String() != "Member" {
			return fmt.Errorf("Find non memeber field in struct reader: %+v", entry)
		}
		// example *Member* entry:
		// {Offset:240737 Tag:Member Children:false
		//  Field:[{Attr:Name Val:stack Class:ClassString}
		//    	   {Attr:DataMemberLoc Val:0 Class:ClassConstant}
		//  	   {Attr:Type Val:241544 Class:ClassReference}
		// 		   {Attr:Attr(10499) Val:false Class: ClassFlag}]}
		m := new(StrtMember)
		m.Offset = uint32(entry.Offset)
		for _, f := range entry.Field {
			if f.Attr.String() == "Name" {
				m.Name = f.Val.(string)
			} else if f.Attr.String() == "DataMemberLoc" {
				m.StrtOffset = f.Val.(int64)
				// calculate previous field's size
				prev.Size = m.StrtOffset - prev.StrtOffset
			}
		}
		s.Members[m.Name] = m
		prev = m
	}
	return nil
}

// PCToFunc convert program counter to symbolic information
func (b *Binary) PCToFunc(addr uint64) *Location {
	loc, ok := b.funcCache[addr]
	if ok {
		return loc
	}

	file, ln, fn := b.SymTable.PCToLine(addr)
	if fn == nil {
		return nil
	}
	loc = &Location{PC: addr, File: file, Line: ln, Func: fn}
	b.funcCache[addr] = loc
	return loc
}
