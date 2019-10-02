package proc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

func toUint64(buf []byte) uint64 {
	return binary.LittleEndian.Uint64(buf)
}

func toUint32(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf)
}

func toFloat64(buf []byte) float64 {
	bits := binary.LittleEndian.Uint64(buf)
	return math.Float64frombits(bits)
}

func toInt32(buf []byte) int32 {
	var res int32
	reader := bytes.NewReader(buf)
	if err := binary.Read(reader, binary.LittleEndian, &res); err != nil {
		return 0
	}
	return res
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
