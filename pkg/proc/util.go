package proc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)


func toUint8(buf []byte) uint8 {
	return uint8(buf[0])
}

func toUint16(buf []byte) uint16 {
	return binary.LittleEndian.Uint16(buf)
}

func toUint32(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf)
}

func toUint64(buf []byte) uint64 {
	return binary.LittleEndian.Uint64(buf)
}


func toFloat64(buf []byte) float64 {
	bits := binary.LittleEndian.Uint64(buf)
	return math.Float64frombits(bits)
}

func toInt8(buf []byte) int8 {
	var res int8
	reader := bytes.NewReader(buf)
	if err := binary.Read(reader, binary.LittleEndian, &res); err != nil {
		return 0
	}
	return res
}

func toInt16(buf []byte) int16 {
	var res int16
	reader := bytes.NewReader(buf)
	if err := binary.Read(reader, binary.LittleEndian, &res); err != nil {
		return 0
	}
	return res
}

func toInt32(buf []byte) int32 {
	var res int32
	reader := bytes.NewReader(buf)
	if err := binary.Read(reader, binary.LittleEndian, &res); err != nil {
		return 0
	}
	return res
}

func toInt64(buf []byte) int64 {
	var res int64
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
