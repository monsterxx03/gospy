package proc

import (
	_ "fmt"
)

const (
	_NumSizeClasses = 67
	// from runtime.mheap.go
	numSpanClasses = _NumSizeClasses << 1
)

// from runtime.sizeclasses.go, put small items(<32KB) into different classes
var class_to_size = [_NumSizeClasses]uint16{0, 8, 16, 32, 48, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224, 240, 256, 288, 320, 352, 384, 416, 448, 480, 512, 576, 640, 704, 768, 896, 1024, 1152, 1280, 1408, 1536, 1792, 2048, 2304, 2688, 3072, 3200, 3456, 4096, 4864, 5376, 6144, 6528, 6784, 6912, 8192, 9472, 9728, 10240, 10880, 12288, 13568, 14336, 16384, 18432, 19072, 20480, 21760, 24576, 27264, 28672, 32768}

type spanClass uint8

func (s spanClass) String() string {
	size := class_to_size[int8(s>>1)]
	return humanateBytes(uint64(size))
}

type MSpan struct {
	common
	Npages     uint64     `name:"npages"`
	SpanClass  spanClass  `name:"spanclass"`
	Sweepgen   uint32     `name:"sweepgen"`
	AllocCount uint16     `name:"allocCount"`
	State      mspanstate `name:"state"`
}

func (s *MSpan) Parse(addr uint64) error {
	return parse(addr, s)
}

type MCentral struct {
	common
	SpanClass spanClass `name:"spanclass"`
	NMalloc   uint64    `name:"nmalloc"`
}

func (c *MCentral) Parse(addr uint64) error {
	return parse(addr, c)
}

// MHeap hold process heap info (runtime/mheap.go:mheap)
type MHeap struct {
	common
	Sweepgen    uint32   `name:"sweepgen"` // used to compare with mspan.sweepgen
	MSpans      []*MSpan `name:"allspans" binStrt:"runtime.mspan"`
	PagesInUse  uint64   `name:"pagesInUse"`  // pages of spans in stats mSpanInUse
	PagesSwept  uint64   `name:"pagesSwept"`  // pages swept this cycle
	NLargeAlloc uint64   `name:"nlargealloc"` // number of large object allocations
	Central     []*MCentral
}

func (h *MHeap) Parse(addr uint64) error {
	bin := h.BinStrt()
	p := h.Process()
	// mheap.central is a anonymous struct, don't how to parse it by reflect so far.
	// hardcode size and parse manually. from: runtime.mheap.go:mheap.central
	arrayLen := int64(numSpanClasses)
	itemSize := bin.Members["central"].Size / arrayLen // 64
	centralSlice := make([]*MCentral, 0, arrayLen)
	// base addr of centray field
	centralAddr := addr + uint64(bin.Members["central"].StrtOffset)
	for i := int64(0); i < arrayLen; i++ {
		mcentral := new(MCentral)
		mcentral.Init(p, p.bin.MCentralStruct)
		maddr := centralAddr + uint64(i*itemSize)
		if err := mcentral.Parse(maddr); err != nil {
			return err
		}
		centralSlice = append(centralSlice, mcentral)
	}
	return parse(addr, h)
}
