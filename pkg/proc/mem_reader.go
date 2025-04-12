package proc

import (
	"io"
)

type ProcessMemReader interface {
	io.ReaderAt
	Close() error
	RuntimeInfo() (*Runtime, error)
	Goroutines() ([]G, error)
	GetGoroutineStackTraceByGoID(goid int64) ([]StackFrame, error)
	Ps() ([]P, error)
	MemStat() (*MemStat, error)
}
