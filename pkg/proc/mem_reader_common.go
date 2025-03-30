package proc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"unsafe"

	bin "github.com/monsterxx03/gospy/pkg/binary"
)

type reader interface {
	io.ReaderAt
	GetBinaryLoader() bin.BinaryLoader
	GetStaticBase() uint64

	// Internal memory reading methods
	readString(addr uint64) (string, error)
	readUint8(addr uint64) (uint8, error)
	readUint32(addr uint64) (uint32, error)
	readUint64(addr uint64) (uint64, error)
	readInt64(addr uint64) (int64, error)
	readSlice(addr uint64) ([]byte, uint64, error)
	readPtrSlice(addr uint64) ([]uint64, error)
}

type commonMemReader struct {
	reader
	pid int
}

func (r *commonMemReader) readBool(addr uint64) (bool, error) {
	v, err := r.readUint8(addr)
	return v != 0, err
}

func (r *commonMemReader) readUint8(addr uint64) (uint8, error) {
	buf := make([]byte, 1)
	if _, err := r.ReadAt(buf, int64(addr)); err != nil {
		return 0, fmt.Errorf("ReadUint8 failed: %w", err)
	}
	return buf[0], nil
}

func (r *commonMemReader) readUint16(addr uint64) (uint16, error) {
	buf := make([]byte, 2)
	if _, err := r.ReadAt(buf, int64(addr)); err != nil {
		return 0, fmt.Errorf("ReadUint16 failed: %w", err)
	}
	return binary.LittleEndian.Uint16(buf), nil
}

func (r *commonMemReader) readUint32(addr uint64) (uint32, error) {
	buf := make([]byte, 4)
	if _, err := r.ReadAt(buf, int64(addr)); err != nil {
		return 0, fmt.Errorf("ReadUint32 failed: %w", err)
	}
	return binary.LittleEndian.Uint32(buf), nil
}

func (r *commonMemReader) readUint64(addr uint64) (uint64, error) {
	buf := make([]byte, 8)
	if _, err := r.ReadAt(buf, int64(addr)); err != nil {
		return 0, fmt.Errorf("ReadUint64 failed: %w", err)
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func (r *commonMemReader) readInt8(addr uint64) (int8, error) {
	v, err := r.readUint8(addr)
	return int8(v), err
}

func (r *commonMemReader) readInt16(addr uint64) (int16, error) {
	v, err := r.readUint16(addr)
	return int16(v), err
}

func (r *commonMemReader) readInt32(addr uint64) (int32, error) {
	v, err := r.readUint32(addr)
	return int32(v), err
}

func (r *commonMemReader) readInt64(addr uint64) (int64, error) {
	v, err := r.readUint64(addr)
	return int64(v), err
}

func (r *commonMemReader) readString(addr uint64) (string, error) {
	header := make([]byte, 16)
	if _, err := r.ReadAt(header, int64(addr)); err != nil {
		return "", fmt.Errorf("failed to read string header: %w", err)
	}

	dataPtr := binary.LittleEndian.Uint64(header[:8])
	strLen := binary.LittleEndian.Uint64(header[8:16])

	if dataPtr == 0 || strLen == 0 {
		return "", nil
	}

	data := make([]byte, strLen)
	if _, err := r.ReadAt(data, int64(dataPtr)); err != nil {
		return "", fmt.Errorf("failed to read string data: %w", err)
	}

	return string(data), nil
}

// readSlice reads a Go slice from memory and returns its raw data and length
var sliceHeaderPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 24)
	},
}

func (r *commonMemReader) readSlice(addr uint64) ([]byte, uint64, error) {
	// Slice header layout (64-bit):
	// 0-7: data pointer
	// 8-15: length
	// 16-23: capacity
	header := sliceHeaderPool.Get().([]byte)
	defer sliceHeaderPool.Put(header)

	if _, err := r.ReadAt(header, int64(addr)); err != nil {
		return nil, 0, fmt.Errorf("failed to read slice header: %w", err)
	}

	dataPtr := binary.LittleEndian.Uint64(header[:8])
	length := binary.LittleEndian.Uint64(header[8:16])
	capacity := binary.LittleEndian.Uint64(header[16:24])

	if dataPtr == 0 || length == 0 {
		return nil, 0, nil
	}

	if length > capacity {
		return nil, 0, fmt.Errorf("invalid slice: length (%d) > capacity (%d)", length, capacity)
	}

	ptrSize := r.GetBinaryLoader().PtrSize()
	data := make([]byte, length*uint64(ptrSize))
	if _, err := r.ReadAt(data, int64(dataPtr)); err != nil {
		return nil, 0, fmt.Errorf("failed to read slice data: %w", err)
	}

	return data, length, nil
}

// readPtrSlice reads a slice of pointers from memory
// readStruct reads a struct from memory into the provided struct pointer
func (r *commonMemReader) readStruct(addr uint64, out interface{}) error {
	size := int(unsafe.Sizeof(out))
	buf := make([]byte, size)
	_, err := r.ReadAt(buf, int64(addr))
	if err != nil {
		return fmt.Errorf("failed to read struct: %w", err)
	}
	return binary.Read(bytes.NewReader(buf), binary.LittleEndian, out)
}

// readArray reads an array from memory in one operation
func (r *commonMemReader) readArray(addr uint64, elementSize, count int) ([]byte, error) {
	buf := make([]byte, elementSize*count)
	if _, err := r.ReadAt(buf, int64(addr)); err != nil {
		return nil, fmt.Errorf("failed to read array: %w", err)
	}
	return buf, nil
}

// readGoroutineBatch reads multiple goroutine structs in one operation
func (r *commonMemReader) readGoroutineBatch(ptrs []uint64, gSize uint64) ([]byte, error) {
	if len(ptrs) == 0 {
		return nil, nil
	}

	// Calculate total size needed
	totalSize := gSize * uint64(len(ptrs))
	buf := make([]byte, totalSize)

	// Read all goroutines in one batch
	for i, ptr := range ptrs {
		if ptr == 0 {
			continue
		}
		if _, err := r.ReadAt(buf[i*int(gSize):(i+1)*int(gSize)], int64(ptr)); err != nil {
			return nil, fmt.Errorf("failed to read goroutine at 0x%x: %w", ptr, err)
		}
	}

	return buf, nil
}

func (r *commonMemReader) readPtrSlice(addr uint64) ([]uint64, error) {
	data, length, err := r.readSlice(addr)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return nil, nil
	}
	ptrSize := r.GetBinaryLoader().PtrSize()
	pointers := make([]uint64, length)

	for i := uint64(0); i < length; i++ {
		offset := i * uint64(ptrSize)
		if ptrSize == 8 {
			pointers[i] = binary.LittleEndian.Uint64(data[offset : offset+8])
		} else {
			pointers[i] = uint64(binary.LittleEndian.Uint32(data[offset : offset+4]))
		}
	}

	return pointers, nil
}
