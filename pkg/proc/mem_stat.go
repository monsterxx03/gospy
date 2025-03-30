package proc

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type MemStat struct {
	// General statistics
	// Alloc      uint64 // bytes allocated and not yet freed
	// TotalAlloc uint64 // bytes allocated (even if freed)
	// Sys        uint64 // bytes obtained from system (sum of XxxSys below)
	// Lookups    uint64 // number of pointer lookups
	// Mallocs    uint64 // number of mallocs
	// Frees      uint64 // number of frees

	// Main allocation heap statistics
	// HeapAlloc    uint64 // bytes allocated and not yet freed (same as Alloc)
	// HeapSys      uint64 // bytes obtained from system
	// HeapIdle     uint64 // bytes in idle spans
	// HeapInuse    uint64 // bytes in non-idle spans
	// HeapReleased uint64 // bytes released to the OS
	// HeapObjects  uint64 // total number of allocated objects

	// Low-level fixed-size structure allocator statistics
	// StackInuse  uint64 // bytes used by stack allocator
	// StackSys    uint64
	// MSpanInuse  uint64 // mspan structures
	// MSpanSys    uint64
	// MCacheInuse uint64 // mcache structures
	// MCacheSys   uint64
	// BuckHashSys uint64 // profiling bucket hash table
	// GCSys       uint64 // GC metadata
	// OtherSys    uint64 // other system allocations

	// Garbage collector statistics
	// NextGC        uint64 // next collection will happen when HeapAlloc ≥ this amount
	LastGC       uint64      `json:"last_gc"` // end time of last collection (unixtimestamp in seconds)
	PauseTotalNs uint64      `json:"pause_total_ns"`
	PauseNs      [256]uint64 `json:"-"` // circular buffer of recent GC pause durations
	PauseEnd     [256]uint64 `json:"-"` // circular buffer of recent GC pause end times
	NumGC        uint32      `json:"num_gc"`
	// GCCPUFraction float64 // fraction of CPU time used by GC
	// EnableGC      bool
	// DebugGC       bool

	// Size classes statistics
	// BySize [61]struct {
	// 	Size    uint32
	// 	Mallocs uint64
	// 	Frees   uint64
	// }
}

func (r *commonMemReader) MemStat() (*MemStat, error) {
	ms := &MemStat{}
	dwarfLoader, err := r.GetBinaryLoader().GetDWARFLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to get DWARF loader: %w", err)
	}

	// Get address of runtime.memstats symbol
	mstatsAddr, err := r.GetBinaryLoader().FindVariableAddress("runtime.memstats")
	if err != nil {
		return nil, fmt.Errorf("failed to find memstats symbol: %w", err)
	}
	baseAddr := r.GetStaticBase() + mstatsAddr

	// Read GC stats fields manually using DWARF info
	var errs []error

	// Read LastGC
	if offset, err := dwarfLoader.GetStructOffset("runtime.mstats", "last_gc_unix"); err == nil {
		if val, err := r.readUint64(baseAddr + offset); err == nil {
			ms.LastGC = val
		} else {
			errs = append(errs, fmt.Errorf("failed to read last_gc: %w", err))
		}
	}

	// Read PauseTotalNs
	if offset, err := dwarfLoader.GetStructOffset("runtime.mstats", "pause_total_ns"); err == nil {
		if val, err := r.readUint64(baseAddr + offset); err == nil {
			ms.PauseTotalNs = val
		} else {
			errs = append(errs, fmt.Errorf("failed to read pause_total_ns: %w", err))
		}
	}

	// Read PauseNs array (batch read)
	if offset, err := dwarfLoader.GetStructOffset("runtime.mstats", "pause_ns"); err == nil {
		arrayAddr := baseAddr + offset
		data, err := r.readArray(arrayAddr, 8, len(ms.PauseNs))
		if err == nil {
			for i := 0; i < len(ms.PauseNs); i++ {
				ms.PauseNs[i] = binary.LittleEndian.Uint64(data[i*8 : (i+1)*8])
			}
		} else {
			errs = append(errs, fmt.Errorf("failed to read pause_ns array: %w", err))
		}
	}

	// Read PauseEnd array (batch read)
	if offset, err := dwarfLoader.GetStructOffset("runtime.mstats", "pause_end"); err == nil {
		arrayAddr := baseAddr + offset
		data, err := r.readArray(arrayAddr, 8, len(ms.PauseEnd))
		if err == nil {
			for i := 0; i < len(ms.PauseEnd); i++ {
				ms.PauseEnd[i] = binary.LittleEndian.Uint64(data[i*8 : (i+1)*8])
			}
		} else {
			errs = append(errs, fmt.Errorf("failed to read pause_end array: %w", err))
		}
	}

	// Read NumGC
	if offset, err := dwarfLoader.GetStructOffset("runtime.mstats", "numgc"); err == nil {
		if val, err := r.readUint32(baseAddr + offset); err == nil {
			ms.NumGC = val
		} else {
			errs = append(errs, fmt.Errorf("failed to read numgc: %w", err))
		}
	}

	// Return partial results even if some fields failed to read
	if len(errs) > 0 {
		return ms, fmt.Errorf("partial read errors: %w", errors.Join(errs...))
	}

	return ms, nil
}
