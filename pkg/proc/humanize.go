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

const (
	ns = 1000
	us = 10000
	ms = 100000
	s  = 1000000
)
const (
	m = s * 60 * (iota + 1)
	h
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

func humanateNS(t uint64) string {
	if t < ns {
		return fmt.Sprintf("%dns", t)
	}
	_t := float64(t)
	if t < us {
		return fmt.Sprintf("%.2fus", _t/ns)
	}
	if t < ms {
		return fmt.Sprintf("%.2fms", _t/us)
	}
	if t < s {
		return fmt.Sprintf("%.2fs", _t/ms)
	}
	if t < m {
		return fmt.Sprintf("%.2fmin", _t/s)
	}
	return fmt.Sprintf("%.2fh", _t/m)
}
