package binary

import (
	"debug/dwarf"
	"debug/elf"
	"debug/gosym"
	"encoding/binary"
	"fmt"
	"os"
	"sync"

	"github.com/golang/glog"
)

const (
	DW_OP_addr = 0x03
)

type Binary struct {
	Path      string
	bin       *elf.File
	addrCache map[string]uint64
	SymTable  *gosym.Table

	// following fields are parsed from binary dwarf during starting
	GoVerAddr   uint64 // parsed vma of runtime.buildVersion
	GStruct     *Strt  // parsed runtime.g struct
	MStruct     *Strt  //parsed runtime.m struct
	AllglenAddr uint64 // parsed vma of runtime.allglen
	AllgsAddr   uint64 // parsed vma of runtime.allgs
}

// Strt is a abstruct struct parsed from dwarf info
type Strt struct {
	Name    string
	Size    int64
	Members map[string]*StrtMember
}

type StrtMember struct {
	Name       string
	Size       int64  // member field size
	Offset     uint32 // offset in binary
	StrtOffset int64  // offset inside struct
}

func Load(pid int, exe string) (*Binary, error) {
	path := fmt.Sprintf("/proc/%d/exe", pid)
	if exe != "" {
		path = exe
	}
	path, err := os.Readlink(path)
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
	return &Binary{Path: path, bin: b, SymTable: symtab, addrCache: make(map[string]uint64)}, nil
}

// Initialize will pre parse some info from elf binary
func (b *Binary) Initialize() error {
	errChan := make(chan error)
	doneChan := make(chan int)
	wg := new(sync.WaitGroup)
	wg.Add(5)
	go func(wg *sync.WaitGroup) {
		g, err := b.GetStruct("runtime.g")
		if err != nil {
			glog.Errorf("Failed to get runtime.g from %s", b.Path)
			errChan <- err
		}
		b.GStruct = g
		wg.Done()
	}(wg)
	go func(wg *sync.WaitGroup) {
		m, err := b.GetStruct("runtime.m")
		if err != nil {
			glog.Errorf("Failed to get runtime.g from %s", b.Path)
			errChan <- err
		}
		b.MStruct = m
		wg.Done()
	}(wg)
	go func(wg *sync.WaitGroup) {
		allglenaddr, err := b.GetVarAddr("runtime.allglen")
		if err != nil {
			glog.Errorf("Failed to get runtime.allglen from %s", b.Path)
			errChan <- err
		}
		b.AllglenAddr = allglenaddr
		wg.Done()
	}(wg)
	go func(wg *sync.WaitGroup) {
		allgsaddr, err := b.GetVarAddr("runtime.allgs")
		if err != nil {
			glog.Errorf("Failed to get runtime.allgs from %s", b.Path)
			errChan <- err
		}
		b.AllgsAddr = allgsaddr
		wg.Done()
	}(wg)
	go func(wg *sync.WaitGroup) {
		goVerAddr, err := b.GetVarAddr("runtime.buildVersion")
		if err != nil {
			glog.Errorf("Failed to get runtime.buildVersion from %s", b.Path)
			errChan <- err
		}
		b.GoVerAddr = goVerAddr
		wg.Done()
	}(wg)
	go func(wg *sync.WaitGroup) {
		wg.Wait()
		doneChan <- 1
	}(wg)
	select {
	case e := <-errChan:
		return e
	case <-doneChan:
		return nil
	}
}

// GetVarAddr will search binary's DWARF info, to find virtual memory address of a global variable
func (b *Binary) GetVarAddr(varName string) (uint64, error) {
	val, ok := b.addrCache[varName]
	if ok {
		return val, nil
	}
	data, err := b.bin.DWARF()
	if err != nil {
		return 0, err
	}
	reader := data.Reader()
	var addr uint64
	for {
		entry, err := reader.Next()
		if err != nil {
			return 0, err
		}
		if entry == nil {
			// reach end
			break
		}
		for _, f := range entry.Field {
			switch f.Val.(type) {
			case string:
				if f.Attr.String() == "Name" && f.Val.(string) == varName {
					// instructions = 1 byte DW_OP type + 8 bytes address
					instructions, ok := entry.Val(dwarf.AttrLocation).([]byte)
					if !ok {
						return 0, fmt.Errorf("Failed to parse %v", entry)
					}
					if len(instructions) != 9 {
						return 0, fmt.Errorf("Read invalid DW_AT_location: %v", instructions)
					}
					if instructions[0] != DW_OP_addr {
						return 0, fmt.Errorf("%s's DW_OP type isn't DW_OP_addr(0x03), don't support to parse: %v", varName, instructions)
					}
					// parse left 8 bytes as virtual memory address
					addr = uint64(binary.LittleEndian.Uint64(instructions[1:]))
					b.addrCache[varName] = addr
					return addr, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("didn't find address for %s", varName)
}

func (b *Binary) GetStruct(name string) (*Strt, error) {
	data, err := b.bin.DWARF()
	if err != nil {
		return nil, err
	}
	result := new(Strt)
	result.Members = make(map[string]*StrtMember)
	reader := data.Reader()
	for {
		entry, err := reader.Next()
		if err != nil {
			return nil, err
		}
		if entry == nil {
			// reach end
			break
		}
		if entry.Tag.String() != "StructType" {
			continue
		}
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
		findTarget := false
		var size int64
		for _, f := range entry.Field {
			if f.Attr.String() == "Name" && f.Val.(string) == name {
				findTarget = true
				result.Name = name
			} else if f.Attr.String() == "ByteSize" {
				size = f.Val.(int64)
			}
		}
		if findTarget {
			result.Size = size
			err := result.parseMembers(reader)
			if err != nil {
				return nil, err
			}
		}
	}
	if result.Name == "" || result.Size == 0 {
		return nil, fmt.Errorf("failed to parse struct %s", name)
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
func (b *Binary) PCToFunc(addr uint64) (file string, ln int, fn *gosym.Func) {
	file, ln, fn = b.SymTable.PCToLine(addr)
	return
}
