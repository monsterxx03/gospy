package proc

import (
	"encoding/binary"
	"fmt"
	"time"
)

func toUint64(buf []byte) uint64 {
	return binary.LittleEndian.Uint64(buf)
}

func toUint32(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf)
}

type perf struct {
	start time.Time
}

func (p *perf) Start() {
	p.start = time.Now()
}

func (p *perf) End() {
	fmt.Println(time.Now().Sub(p.start))
}
