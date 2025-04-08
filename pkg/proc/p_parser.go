package proc

import (
	"encoding/binary"
	"fmt"
	bin "github.com/monsterxx03/gospy/pkg/binary"
)

func (r *commonMemReader) Ps() ([]P, error) {
	// Get the address of runtime.allp symbol
	allpAddr, err := r.GetBinaryLoader().FindVariableAddress("runtime.allp")
	if err != nil {
		return nil, fmt.Errorf("find allp symbol: %w", err)
	}

	// Read the slice of P pointers from runtime.allp
	ptrs, err := r.readPtrSlice(r.GetStaticBase() + allpAddr)
	if err != nil {
		return nil, fmt.Errorf("read allp slice: %w", err)
	}

	// Get size of runtime.p struct
	dwarfLoader, err := r.GetBinaryLoader().GetDWARFLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to get DWARF loader: %w", err)
	}
	pSize, err := dwarfLoader.GetStructSize("runtime.p")
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime.p size: %w", err)
	}

	// Read all P structs in one batch
	pData, err := r.readGoroutineBatch(ptrs, pSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read P batch: %w", err)
	}

	// Parse each P from the batch data
	ps := make([]P, 0, len(ptrs))
	for i, ptr := range ptrs {
		if ptr == 0 {
			continue
		}
		p, err := r.parsePFromBatch(pData[i*int(pSize):(i+1)*int(pSize)], ptr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse P at 0x%x: %w", ptr, err)
		}
		ps = append(ps, p)
	}

	return ps, nil
}

func (r *commonMemReader) parsePFromBatch(data []byte, pAddr uint64) (P, error) {
	p := P{Address: pAddr}
	dwarfLoader, err := r.GetBinaryLoader().GetDWARFLoader()
	if err != nil {
		return p, fmt.Errorf("failed to get DWARF loader: %w", err)
	}

	// Parse ID
	idOffset, err := dwarfLoader.GetStructOffset("runtime.p", "id")
	if err != nil {
		return p, fmt.Errorf("failed to get id offset: %w", err)
	}
	p.ID = int32(binary.LittleEndian.Uint32(data[idOffset:]))

	// Parse status
	statusOffset, err := dwarfLoader.GetStructOffset("runtime.p", "status")
	if err != nil {
		return p, fmt.Errorf("failed to get status offset: %w", err)
	}
	p.Status = binary.LittleEndian.Uint32(data[statusOffset:]))
	p.StatusString = parsePStatus(p.Status)

	// Parse mcache
	mcacheOffset, err := dwarfLoader.GetStructOffset("runtime.p", "mcache")
	if err != nil {
		return p, fmt.Errorf("failed to get mcache offset: %w", err)
	}
	p.MCache = binary.LittleEndian.Uint64(data[mcacheOffset:]))

	// Parse schedtick
	schedtickOffset, err := dwarfLoader.GetStructOffset("runtime.p", "schedtick")
	if err != nil {
		return p, fmt.Errorf("failed to get schedtick offset: %w", err)
	}
	p.SchedTick = binary.LittleEndian.Uint32(data[schedtickOffset:]))

	// Parse runq size
	runqSizeOffset, err := dwarfLoader.GetStructOffset("runtime.p", "runqsize")
	if err != nil {
		return p, fmt.Errorf("failed to get runqsize offset: %w", err)
	}
	p.RunqSize = int32(binary.LittleEndian.Uint32(data[runqSizeOffset:]))

	return p, nil
}

func parsePStatus(status uint32) string {
	switch status {
	case _Pidle:
		return "idle"
	case _Prunning:
		return "running"
	case _Psyscall:
		return "syscall"
	case _Pgcstop:
		return "gcstop"
	case _Pdead:
		return "dead"
	default:
		return fmt.Sprintf("unknown(%d)", status)
	}
}
