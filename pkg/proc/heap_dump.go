package proc

import (
	"encoding/binary"
	"fmt"

	"debug/dwarf"
	"sort"
	"strings"

	bin "github.com/monsterxx03/gospy/pkg/binary"
)

// HeapObject represents a single allocated object on the heap.
type HeapObject struct {
	Addr  uint64 `json:"addr"`
	Size  uint64 `json:"size"`
	Type  string `json:"type"`
	Guess bool   `json:"guess"` // whether Type is heuristic guess
}

// HeapTypeStat is aggregated statistics for a type
type HeapTypeStat struct {
	Type        string `json:"type"`
	Objects     uint64 `json:"objects"`
	Bytes       uint64 `json:"bytes"`
	IsEstimated bool   `json:"estimated"`
}

// HeapScanOptions controls type distribution scan
type HeapScanOptions struct {
	// Sample every Nth span to speed up; 1 means scan all
	SpanSample int
	// Ignore objects smaller than this many bytes
	MinObjectSize uint64
	// Limit number of spans to scan (0 = no limit)
	MaxSpans int
	// Return only the top N types by bytes (0 = all)
	TopN int
}

// HeapObjects walks the runtime heap metadata to enumerate allocated objects.
// This is a best-effort implementation based on runtime data structures and DWARF.
// It may need updates across Go versions.
func (r *commonMemReader) HeapObjects(limit int) ([]HeapObject, error) {
	dwarfLoader, err := r.GetBinaryLoader().GetDWARFLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to get DWARF loader: %w", err)
	}

	// Locate runtime.mheap_ symbol (global heap state)
	// In the runtime, the symbol is named "runtime.mheap_".
	mheapAddr, err := r.GetBinaryLoader().FindVariableAddress("runtime.mheap_")
	if err != nil {
		return nil, fmt.Errorf("failed to find runtime.mheap_: %w", err)
	}
	mheapBase := r.GetStaticBase() + mheapAddr

	// The mheap has arenas metadata; we can read all arenas via allArenas slice.
	// Use DWARF to find offsets.
	allArenasOff, err := dwarfLoader.GetStructOffset("runtime.mheap", "allArenas")
	if err != nil {
		return nil, fmt.Errorf("get mheap.allArenas offset: %w", err)
	}

	// Read slice of arena indexes: []arenaIdx (underlying is uint)
	arenasSliceData, arenasLen, err := r.readSlice(mheapBase + allArenasOff)
	if err != nil {
		return nil, fmt.Errorf("read mheap.allArenas slice: %w", err)
	}
	if arenasLen == 0 || len(arenasSliceData) == 0 {
		return nil, nil
	}

	// heapArena metadata is in runtime.heap_.arenas (2-level), but we can use allspans as fallback.
	// Prefer walking allspans, filter in-use spans and enumerate objects.
	allspansOff, err := dwarfLoader.GetStructOffset("runtime.mheap", "allspans")
	if err != nil {
		return nil, fmt.Errorf("get mheap.allspans offset: %w", err)
	}

	spanPtrs, err := r.readPtrSlice(mheapBase + allspansOff)
	if err != nil {
		return nil, fmt.Errorf("read mheap.allspans slice: %w", err)
	}
	if len(spanPtrs) == 0 {
		return nil, nil
	}

	// Get mspan size and offsets we need
	mspanSize, err := dwarfLoader.GetStructSize("runtime.mspan")
	if err != nil {
		return nil, fmt.Errorf("get runtime.mspan size: %w", err)
	}
	// Field names vary by versions. Try common ones.
	baseOff, err := firstOffset(dwarfLoader, "runtime.mspan", []string{"base", "startAddr"})
	if err != nil {
		return nil, err
	}
	nelemsOff, err := firstOffset(dwarfLoader, "runtime.mspan", []string{"nelems"})
	if err != nil {
		return nil, err
	}
	elemsizeOff, err := firstOffset(dwarfLoader, "runtime.mspan", []string{"elemsize"})
	if err != nil {
		return nil, err
	}
	stateOff, _ := firstOffset(dwarfLoader, "runtime.mspan", []string{"state"}) // optional
	allocBitsOff, _ := firstOffset(dwarfLoader, "runtime.mspan", []string{"allocBits"})

	// Batch read spans for speed
	data, err := r.readPtrBatch(spanPtrs, mspanSize)
	if err != nil {
		return nil, fmt.Errorf("read mspan batch: %w", err)
	}

	// Build a quick map from size -> candidate type names (structs/arrays) using DWARF.
	sizeToTypes := buildSizeToTypesMap(dwarfLoader)
	// Build runtime type pointer -> type name map from binary symbols (type.*)
	typeAddrToName := r.buildTypeSymbolMap()

	var out []HeapObject
	appendObj := func(addr, size uint64) bool {
		if tname, ok := r.resolveTypeByHeader(typeAddrToName, addr); ok {
			out = append(out, HeapObject{Addr: addr, Size: size, Type: tname, Guess: false})
			return limit > 0 && len(out) >= limit
		}
		objType, guessed := matchTypeBySize(sizeToTypes, size)
		out = append(out, HeapObject{Addr: addr, Size: size, Type: objType, Guess: guessed})
		return limit > 0 && len(out) >= limit
	}

	ptrSize := r.GetBinaryLoader().PtrSize()

	for i, ptr := range spanPtrs {
		if ptr == 0 {
			continue
		}
		sd := data[i*int(mspanSize) : (i+1)*int(mspanSize)]

		// Check state if available: we only want in-use spans
		if stateOff != 0 {
			st := sd[int(stateOff)]
			// mSpanInUse == 1 in recent versions
			if st != 1 {
				continue
			}
		}

		base := binary.LittleEndian.Uint64(sd[int(baseOff):])
		if base == 0 {
			continue
		}
		nelems := binary.LittleEndian.Uint32(sd[int(nelemsOff):])
		elemsize := binary.LittleEndian.Uint32(sd[int(elemsizeOff):])
		if nelems == 0 || elemsize == 0 {
			continue
		}

		// If allocBits present, use it to filter allocated slots
		var allocBitsAddr uint64
		if allocBitsOff != 0 {
			if ptrSize == 8 {
				allocBitsAddr = binary.LittleEndian.Uint64(sd[int(allocBitsOff):])
			} else {
				allocBitsAddr = uint64(binary.LittleEndian.Uint32(sd[int(allocBitsOff):]))
			}
		}

		// Enumerate objects in the span
		objSize := uint64(elemsize)
		for j := uint32(0); j < nelems; j++ {
			objAddr := base + uint64(j)*objSize
			allocated := true
			if allocBitsAddr != 0 {
				byteIdx := j / 8
				bitMask := byte(1 << (j % 8))
				b, err := r.readUint8(allocBitsAddr + uint64(byteIdx))
				if err != nil {
					continue
				}
				allocated = (b & bitMask) != 0
			}
			if allocated {
				if appendObj(objAddr, objSize) {
					return out, nil
				}
			}
		}
	}
	return out, nil
}

func firstOffset(d bin.DWARFLoader, structName string, candidates []string) (uint64, error) {
	var errs []error
	for _, f := range candidates {
		off, err := d.GetStructOffset(structName, f)
		if err == nil {
			return off, nil
		}
		errs = append(errs, err)
	}
	return 0, fmt.Errorf("no offset found in %s for fields %v: %v", structName, candidates, errs)
}

// buildSizeToTypesMap gathers a map size -> list of type names with that size.
// We only index named struct/array types to reduce noise.
func buildSizeToTypesMap(d bin.DWARFLoader) map[uint64][]string {
	m := make(map[uint64][]string)
	data, err := d.Raw()
	if err != nil || data == nil {
		return m
	}
	r := data.Reader()
	for {
		e, err := r.Next()
		if err != nil || e == nil {
			break
		}
		switch e.Tag {
		case dwarf.TagStructType, dwarf.TagArrayType:
			name, _ := e.Val(dwarf.AttrName).(string)
			// Only index named types to avoid anonymous noise
			if name == "" {
				continue
			}
			if sz, ok := e.Val(dwarf.AttrByteSize).(int64); ok && sz > 0 {
				m[uint64(sz)] = append(m[uint64(sz)], name)
			}
		}
	}
	return m
}

// matchTypeBySize returns a type name and whether it is a guess (true if multiple candidates or empty).
func matchTypeBySize(m map[uint64][]string, size uint64) (string, bool) {
	names := m[size]
	if len(names) == 1 {
		return names[0], false
	}
	if len(names) > 1 {
		// ambiguous; return first as guess
		return names[0], true
	}
	return "", true
}

// buildTypeSymbolMap collects addresses of runtime type descriptors by scanning symbols like type.*
// Returns map[typeAddr]->typeName.
func (r *commonMemReader) buildTypeSymbolMap() map[uint64]string {
	result := make(map[uint64]string)
	syms, err := r.GetBinaryLoader().GetSymbols()
	if err != nil {
		return result
	}
	for name, addr := range syms {
		// Go puts named type symbols in the form "type.<full>" in the binary symbol table
		if strings.HasPrefix(name, "type.") {
			// stored addresses are relative to static base in our readers
			result[r.GetStaticBase()+addr] = strings.TrimPrefix(name, "type.")
		}
	}
	return result
}

// resolveTypeByHeader attempts to read the per-object type pointer from the object header
// (for pointerful objects on modern Go) and map it to a type name using buildTypeSymbolMap.
func (r *commonMemReader) resolveTypeByHeader(typeAddrToName map[uint64]string, objAddr uint64) (string, bool) {
	// Newer Go versions place an 8-byte type pointer header just before objects
	// with pointers (and sometimes even for pointer-free under certain settings).
	// Try reading at objAddr-8; if it points to a known type symbol, accept it.
	hdrAddr := objAddr - 8
	// Avoid underflow
	if objAddr < 8 {
		return "", false
	}
	// Read candidate type pointer
	tptr, err := r.readUint64(hdrAddr)
	if err != nil || tptr == 0 {
		return "", false
	}
	if name, ok := typeAddrToName[tptr]; ok {
		return name, true
	}
	return "", false
}

// HeapTypes aggregates type distribution with options for speed.
func (r *commonMemReader) HeapTypes(opts HeapScanOptions) ([]HeapTypeStat, error) {
	dwarfLoader, err := r.GetBinaryLoader().GetDWARFLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to get DWARF loader: %w", err)
	}
	mheapAddr, err := r.GetBinaryLoader().FindVariableAddress("runtime.mheap_")
	if err != nil {
		return nil, fmt.Errorf("failed to find runtime.mheap_: %w", err)
	}
	mheapBase := r.GetStaticBase() + mheapAddr

	allspansOff, err := dwarfLoader.GetStructOffset("runtime.mheap", "allspans")
	if err != nil {
		return nil, fmt.Errorf("get mheap.allspans offset: %w", err)
	}
	spanPtrs, err := r.readPtrSlice(mheapBase + allspansOff)
	if err != nil {
		return nil, fmt.Errorf("read mheap.allspans slice: %w", err)
	}
	if len(spanPtrs) == 0 {
		return nil, nil
	}

	mspanSize, err := dwarfLoader.GetStructSize("runtime.mspan")
	if err != nil {
		return nil, fmt.Errorf("get runtime.mspan size: %w", err)
	}
	baseOff, err := firstOffset(dwarfLoader, "runtime.mspan", []string{"base", "startAddr"})
	if err != nil {
		return nil, err
	}
	nelemsOff, err := firstOffset(dwarfLoader, "runtime.mspan", []string{"nelems"})
	if err != nil {
		return nil, err
	}
	elemsizeOff, err := firstOffset(dwarfLoader, "runtime.mspan", []string{"elemsize"})
	if err != nil {
		return nil, err
	}
	stateOff, _ := firstOffset(dwarfLoader, "runtime.mspan", []string{"state"})
	allocBitsOff, _ := firstOffset(dwarfLoader, "runtime.mspan", []string{"allocBits"})

	// Prepare helpers
	sizeToTypes := buildSizeToTypesMap(dwarfLoader)
	typeAddrToName := r.buildTypeSymbolMap()
	ptrSize := r.GetBinaryLoader().PtrSize()

	// Sampling controls
	spanSample := opts.SpanSample
	if spanSample <= 0 {
		spanSample = 1
	}
	maxSpans := opts.MaxSpans
	scanned := 0

	// Aggregates
	typeToStat := make(map[string]*HeapTypeStat)

	// Batch-read spans in chunks to reduce ReadAt calls
	data, err := r.readPtrBatch(spanPtrs, mspanSize)
	if err != nil {
		return nil, fmt.Errorf("read mspan batch: %w", err)
	}

	for i, ptr := range spanPtrs {
		if ptr == 0 {
			continue
		}
		if i%spanSample != 0 {
			continue
		}
		if maxSpans > 0 && scanned >= maxSpans {
			break
		}
		scanned++

		sd := data[i*int(mspanSize) : (i+1)*int(mspanSize)]

		if stateOff != 0 {
			st := sd[int(stateOff)]
			if st != 1 { // only in-use spans
				continue
			}
		}
		base := binary.LittleEndian.Uint64(sd[int(baseOff):])
		if base == 0 {
			continue
		}
		nelems := binary.LittleEndian.Uint32(sd[int(nelemsOff):])
		elemsize := binary.LittleEndian.Uint32(sd[int(elemsizeOff):])
		if nelems == 0 || elemsize == 0 {
			continue
		}
		objSize := uint64(elemsize)
		if opts.MinObjectSize > 0 && objSize < opts.MinObjectSize {
			continue
		}

		var allocBitsAddr uint64
		if allocBitsOff != 0 {
			if ptrSize == 8 {
				allocBitsAddr = binary.LittleEndian.Uint64(sd[int(allocBitsOff):])
			} else {
				allocBitsAddr = uint64(binary.LittleEndian.Uint32(sd[int(allocBitsOff):]))
			}
		}

		for j := uint32(0); j < nelems; j++ {
			if allocBitsAddr != 0 {
				byteIdx := j / 8
				bitMask := byte(1 << (j % 8))
				b, err := r.readUint8(allocBitsAddr + uint64(byteIdx))
				if err != nil {
					continue
				}
				if (b & bitMask) == 0 {
					continue
				}
			}
			objAddr := base + uint64(j)*objSize
			// Fast exact type (header) or fallback size match
			tname, ok := r.resolveTypeByHeader(typeAddrToName, objAddr)
			guessed := false
			if !ok {
				tname, guessed = matchTypeBySize(sizeToTypes, objSize)
				if tname == "" {
					tname = "<unknown>"
					guessed = true
				}
			}
			st := typeToStat[tname]
			if st == nil {
				st = &HeapTypeStat{Type: tname}
				typeToStat[tname] = st
			}
			st.Objects++
			st.Bytes += objSize
			st.IsEstimated = st.IsEstimated || guessed
		}
	}

	// Convert and sort by Bytes desc; apply TopN
	stats := make([]HeapTypeStat, 0, len(typeToStat))
	for _, v := range typeToStat {
		stats = append(stats, *v)
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Bytes > stats[j].Bytes })
	if opts.TopN > 0 && len(stats) > opts.TopN {
		stats = stats[:opts.TopN]
	}
	return stats, nil
}
