package proc

import (
	"encoding/binary"
)

func toUint64(buf []byte) uint64 {
	return binary.LittleEndian.Uint64(buf)
}

func toUint32(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf)
}
