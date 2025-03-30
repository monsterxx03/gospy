//go:build linux

package proc

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"

	bin "github.com/monsterxx03/gospy/pkg/binary"
)

type linuxMemReader struct {
	commonMemReader
	pid        int
	fd         *os.File
	bin        bin.BinaryLoader
	staticBase uint64
}

func NewProcessMemReader(pid int, binPath string) (ProcessMemReader, error) {
	loader := bin.NewBinaryLoader()
	var err error
	if binPath != "" {
		err = loader.Load(binPath)
	} else {
		err = loader.LoadByPid(pid)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load binary: %w", err)
	}

	entryPoint, err := getEntryPoint(pid, loader)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate static base: %w", err)
	}

	fd, err := os.Open(fmt.Sprintf("/proc/%d/mem", pid))
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/%d/mem: %w", pid, err)
	}

	lr := &linuxMemReader{
		pid:        pid,
		fd:         fd,
		bin:        loader,
		staticBase: entryPoint - loader.GetFile().(*elf.File).Entry,
	}
	cr := commonMemReader{reader: lr, pid: pid}
	lr.commonMemReader = cr
	return lr, nil
}

func getEntryPoint(pid int, loader bin.BinaryLoader) (uint64, error) {
	auxvPath := filepath.Join("/proc", fmt.Sprintf("%d", pid), "auxv")
	data, err := os.ReadFile(auxvPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read auxv: %w", err)
	}

	return parseAuxvEntry(data, loader.(*bin.LinuxBinaryLoader).PtrSize()), nil
}

func parseAuxvEntry(data []byte, ptrSize int) uint64 {
	rd := bytes.NewReader(data)
	for {
		tag, err := readUintRaw(rd, binary.LittleEndian, ptrSize)
		if err != nil {
			return 0
		}
		val, err := readUintRaw(rd, binary.LittleEndian, ptrSize)
		if err != nil {
			return 0
		}
		switch tag {
		case _AT_ENTRY:
			return val
		case _AT_NULL:
			return 0
		}
	}
}

func readUintRaw(rd io.Reader, order binary.ByteOrder, ptrSize int) (uint64, error) {
	switch ptrSize {
	case 4:
		var v uint32
		if err := binary.Read(rd, order, &v); err != nil {
			return 0, err
		}
		return uint64(v), nil
	case 8:
		var v uint64
		if err := binary.Read(rd, order, &v); err != nil {
			return 0, err
		}
		return v, nil
	default:
		return 0, fmt.Errorf("unsupported pointer size: %d", ptrSize)
	}
}

const (
	_AT_PHDR  = 3
	_AT_ENTRY = 9
	_AT_BASE  = 7
	_AT_NULL  = 0
)

func (r *linuxMemReader) ReadAt(p []byte, off int64) (n int, err error) {
	return r.fd.ReadAt(p, off)
}

func (r *linuxMemReader) Close() error {
	return r.fd.Close()
}

func (r *linuxMemReader) GetBinaryLoader() bin.BinaryLoader {
	return r.bin
}

func (r *linuxMemReader) GetStaticBase() uint64 {
	return r.staticBase
}
