package proc

import (
	"encoding/binary"
	"fmt"

	bin "github.com/monsterxx03/gospy/pkg/binary"
)

func (r *commonMemReader) parseStackInfo(g *G, data []byte, gAddr uint64, dwarfLoader bin.DWARFLoader) error {
	// Get stack struct offset
	stackOffset, err := dwarfLoader.GetStructOffset("runtime.g", "stack")
	if err != nil {
		return fmt.Errorf("failed to get stack offset: %w", err)
	}

	// Get stack.lo offset
	loOffset, err := dwarfLoader.GetStructOffset("runtime.stack", "lo")
	if err != nil {
		return fmt.Errorf("failed to get stack.lo offset: %w", err)
	}

	// Get stack.hi offset
	hiOffset, err := dwarfLoader.GetStructOffset("runtime.stack", "hi")
	if err != nil {
		return fmt.Errorf("failed to get stack.hi offset: %w", err)
	}

	// Read stack.lo from batch data
	lo := binary.LittleEndian.Uint64(data[stackOffset+loOffset:])

	// Read stack.hi from batch data
	hi := binary.LittleEndian.Uint64(data[stackOffset+hiOffset:])

	g.Stack = Stack{Lo: lo, Hi: hi}
	return nil
}

// parseSchedInfo parses the scheduling information from batch data
func (r *commonMemReader) parseSchedInfo(g *G, data []byte, gAddr uint64, dwarfLoader bin.DWARFLoader) error {
	// Get sched struct offset
	schedOffset, err := dwarfLoader.GetStructOffset("runtime.g", "sched")
	if err != nil {
		return fmt.Errorf("failed to get sched offset: %w", err)
	}

	// Get pc offset
	pcOffset, err := dwarfLoader.GetStructOffset("runtime.gobuf", "pc")
	if err != nil {
		return fmt.Errorf("failed to get pc offset: %w", err)
	}

	// Get sp offset
	spOffset, err := dwarfLoader.GetStructOffset("runtime.gobuf", "sp")
	if err != nil {
		return fmt.Errorf("failed to get sp offset: %w", err)
	}

	// Read pc from batch data
	pc := binary.LittleEndian.Uint64(data[schedOffset+pcOffset:])

	// Read sp from batch data
	sp := binary.LittleEndian.Uint64(data[schedOffset+spOffset:])

	g.Sched = Sched{PC: pc, SP: sp}

	// Get function name if PC is valid
	if pc != 0 {
		funcLoc := r.GetBinaryLoader().PCToFuncLoc(pc - r.GetStaticBase())
		if funcLoc != nil {
			g.FuncName = funcLoc.Desc()
		}
	}

	return nil
}

// readGoroutineField is a helper to read any goroutine field with proper offset handling
func readGoroutineField[T any](
	r *commonMemReader,
	dwarfLoader bin.DWARFLoader,
	baseAddr uint64,
	structName string,
	fieldName string,
	readerFunc func(uint64) (T, error),
) (T, error) {
	var zero T

	offset, err := dwarfLoader.GetStructOffset(structName, fieldName)
	if err != nil {
		return zero, fmt.Errorf("failed to get %s offset: %w", fieldName, err)
	}

	return readerFunc(baseAddr + offset)
}

// parseStatus converts the raw status value to a human-readable string
func (r *commonMemReader) parseStatus(status uint32) string {
	baseStatus := status &^ _Gscan
	scanBit := status & _Gscan

	if name, ok := gStatusMap[baseStatus]; ok {
		if scanBit != 0 {
			if scanState, ok := gStatusMap[status]; ok {
				return scanState
			}
			return "scanunknown"
		}
		return name
	}
	return fmt.Sprintf("unknown(%d)", status)
}

// parseWaitReason parses the wait reason for a waiting goroutine from batch data
func (r *commonMemReader) parseWaitReason(data []byte, gAddr uint64, dwarfLoader bin.DWARFLoader) string {
	rt, err := r.RuntimeInfo()
	if err != nil {
		return fmt.Sprintf("version_error(%v)", err)
	}

	waitReasonOffset, err := dwarfLoader.GetStructOffset("runtime.g", "waitreason")
	if err != nil {
		return fmt.Sprintf("offset_error(%v)", err)
	}
	reason := data[waitReasonOffset]

	if name, ok := registry.GetWaitReasonMap(rt.GoVersion)[reason]; ok {
		return name
	}
	return fmt.Sprintf("unknown(%d)", reason)
}

func (r *commonMemReader) Goroutines() ([]G, error) {
	// Get the address of runtime.allgs symbol
	allgsAddr, err := r.GetBinaryLoader().FindVariableAddress("runtime.allgs")
	if err != nil {
		return nil, fmt.Errorf("find allgs symbol: %w", err)
	}

	// Read the slice of G pointers from runtime.allgs
	ptrs, err := r.readPtrSlice(r.GetStaticBase() + allgsAddr)
	if err != nil {
		return nil, fmt.Errorf("read allgs slice: %w", err)
	}

	// Get size of runtime.g struct
	dwarfLoader, err := r.GetBinaryLoader().GetDWARFLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to get DWARF loader: %w", err)
	}
	gSize, err := dwarfLoader.GetStructSize("runtime.g")
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime.g size: %w", err)
	}

	// Read all goroutine structs in one batch
	gData, err := r.readPtrBatch(ptrs, gSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read goroutine batch: %w", err)
	}

	// Parse each goroutine from the batch data
	gs := make([]G, 0, len(ptrs))
	for i, ptr := range ptrs {
		if ptr == 0 {
			continue
		}
		g, err := r.parseGoroutineFromBatch(gData[i*int(gSize):(i+1)*int(gSize)], ptr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse goroutine at 0x%x: %w", ptr, err)
		}
		gs = append(gs, g)
	}

	return gs, nil
}

// parseBasicInfoFromBatch parses basic goroutine info from pre-read batch data
func (r *commonMemReader) parseBasicInfoFromBatch(g *G, data []byte, gAddr uint64, dwarfLoader bin.DWARFLoader) error {
	// Parse Goid
	goidOffset, err := dwarfLoader.GetStructOffset("runtime.g", "goid")
	if err != nil {
		return fmt.Errorf("failed to get goid offset: %w", err)
	}
	g.Goid = int64(binary.LittleEndian.Uint64(data[goidOffset:]))

	// Parse status
	statusOffset, err := dwarfLoader.GetStructOffset("runtime.g", "atomicstatus")
	if err != nil {
		return fmt.Errorf("failed to get status offset: %w", err)
	}
	g.AtomicStatus = binary.LittleEndian.Uint32(data[statusOffset:])
	g.Status = r.parseStatus(g.AtomicStatus)

	// Parse wait reason if needed
	if g.Status == "waiting" {
		g.WaitReason = r.parseWaitReason(data, gAddr, dwarfLoader)
	}

	// Parse startpc (goroutine's starting function)
	startpcOffset, err := dwarfLoader.GetStructOffset("runtime.g", "startpc")
	if err != nil {
		return fmt.Errorf("failed to get startpc offset: %w", err)
	}
	g.StartPC = binary.LittleEndian.Uint64(data[startpcOffset:])

	// Get start function name if startpc is valid
	if g.StartPC != 0 {
		funcLoc := r.GetBinaryLoader().PCToFuncLoc(g.StartPC - r.GetStaticBase())
		if funcLoc != nil {
			g.StartFuncName = funcLoc.Desc()
		}
	}

	return nil
}

// parseGoroutineFromBatch parses a goroutine from pre-read batch data
func (r *commonMemReader) parseGoroutineFromBatch(data []byte, gAddr uint64) (G, error) {
	g := G{Address: gAddr}
	dwarfLoader, err := r.GetBinaryLoader().GetDWARFLoader()
	if err != nil {
		return g, fmt.Errorf("failed to get DWARF loader: %w", err)
	}

	// Parse basic info from batch data
	if err := r.parseBasicInfoFromBatch(&g, data, gAddr, dwarfLoader); err != nil {
		return g, err
	}

	// Parse stack info from batch data
	if err := r.parseStackInfo(&g, data, gAddr, dwarfLoader); err != nil {
		return g, err
	}

	// Parse scheduling info from batch data
	if err := r.parseSchedInfo(&g, data, gAddr, dwarfLoader); err != nil {
		return g, err
	}

	return g, nil
}
