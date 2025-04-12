package proc

import (
	"io"
)

type ProcessMemReader interface {
	io.ReaderAt
	Close() error
	RuntimeInfo() (*Runtime, error)
	// ai! add showDead option to Goroutines func, by default, this func will filter goroutines in dead status
	Goroutines() ([]G, error)
	GetGoroutineStackTraceByGoID(goid int64) ([]StackFrame, error)
	Ps() ([]P, error)
	MemStat() (*MemStat, error)
}
