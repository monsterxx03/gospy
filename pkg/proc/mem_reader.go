package proc

import (
	"io"
)

type ProcessMemReader interface {
	io.ReaderAt
	Close() error
	RuntimeInfo() (*Runtime, error)
	Goroutines(showDead bool) ([]G, error)
	GetGoroutineStackTraceByGoID(goid int64) ([]StackFrame, error)
	Ps() ([]P, error)
	MemStat() (*MemStat, error)
	// HeapObjects returns allocated heap objects (address and size). If limit > 0, stops after collecting that many objects.
	HeapObjects(limit int) ([]HeapObject, error)
	// HeapTypes returns aggregated type distribution. Options control speed/accuracy tradeoffs.
	HeapTypes(opts HeapScanOptions) ([]HeapTypeStat, error)
}
