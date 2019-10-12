package proc

import (
	"fmt"
)

const (
	_ = 1 << (iota * 10)
	b
	kb
	mb
	gb
)

func humanateBytes(s uint64) string {
	if s < b {
		return fmt.Sprintf("%dB", s)
	}
	if s < kb {
		return fmt.Sprintf("%.2fKB", float64(s)/b)
	}
	if s < mb {
		return fmt.Sprintf("%.2fMB", float64(s)/kb)
	}
	return fmt.Sprintf("%.2fGB", float64(s)/mb)
}
