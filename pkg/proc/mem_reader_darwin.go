//go:build darwin

package proc

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework CoreServices

#include <mach/mach.h>
#include <mach/mach_vm.h>
#include <stdlib.h>
*/
import "C"
import (
	"debug/macho"
	"fmt"
	"unsafe"

	bin "github.com/monsterxx03/gospy/pkg/binary"
)

type darwinMemReader struct {
	commonMemReader
	pid        int
	task       C.task_t
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

	var task C.task_t
	if kRet := C.task_for_pid(C.mach_task_self_, C.int(pid), &task); kRet != C.KERN_SUCCESS {
		return nil, fmt.Errorf("task_for_pid err: %s", machErrorToString(kRet))
	}

	dr := &darwinMemReader{
		pid:  pid,
		task: task,
		bin:  loader,
	}
	cr := commonMemReader{reader: dr, pid: pid}
	dr.commonMemReader = cr

	dr.staticBase, err = dr.getStaticBase()
	if err != nil {
		return nil, fmt.Errorf("failed to get static base: %w", err)
	}
	return dr, nil
}

func machErrorToString(err C.kern_return_t) string {
	cStr := C.mach_error_string(err)
	return C.GoString(cStr)
}

func (r *darwinMemReader) getEntryPoint() (uint64, error) {
	var task C.task_t
	if kRet := C.task_for_pid(C.mach_task_self_, C.pid_t(r.pid), &task); kRet != C.KERN_SUCCESS {
		return 0, fmt.Errorf("task_for_pid failed: %s", machErrorToString(kRet))
	}

	var address C.mach_vm_address_t = 0
	var size C.mach_vm_size_t
	var info C.vm_region_basic_info_data_64_t
	var infoCnt C.mach_msg_type_number_t = C.VM_REGION_BASIC_INFO_COUNT_64
	var objectName C.mach_port_t
	// find first vmm region
	if kRet := C.mach_vm_region(
		task,
		&address,
		&size,
		C.VM_REGION_BASIC_INFO_64,
		(C.vm_region_info_t)(unsafe.Pointer(&info)),
		&infoCnt,
		&objectName,
	); kRet != C.KERN_SUCCESS {
		return 0, fmt.Errorf("mach_vm_region failed: %s", machErrorToString(kRet))
	}
	defer C.mach_port_deallocate(C.mach_task_self_, task)
	return uint64(address), nil
}

func (r *darwinMemReader) getStaticBase() (uint64, error) {
	machoOff := uint64(0x100000000)
	for _, ld := range r.bin.GetFile().(*macho.File).Loads {
		if seg, _ := ld.(*macho.Segment); seg != nil {
			if seg.Name == "__TEXT" {
				machoOff = seg.Addr
				break
			}
		}
	}
	entryPoint, err := r.getEntryPoint()
	if err != nil {
		return 0, fmt.Errorf("getStaticBase: %w", err)
	}
	return entryPoint - machoOff, nil
}

func (r *darwinMemReader) GetStaticBase() uint64 {
	return r.staticBase
}

func (r *darwinMemReader) ReadAt(p []byte, off int64) (n int, err error) {
	var (
		data  C.vm_offset_t
		count C.mach_msg_type_number_t
	)

	if kernReturn := C.mach_vm_read(
		r.task,
		C.mach_vm_address_t(off),
		C.mach_vm_size_t(len(p)),
		&data,
		&count,
	); kernReturn != C.KERN_SUCCESS {
		return 0, fmt.Errorf("mach_vm_read failed: %s", machErrorToString(kernReturn))
	}
	defer C.vm_deallocate(C.mach_task_self_, data, C.vm_size_t(count))

	if count == 0 {
		return 0, nil
	}

	cBuf := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(data))), int(count))
	return copy(p, cBuf), nil
}

func (r *darwinMemReader) Close() error {
	if r.task != C.MACH_PORT_NULL {
		C.mach_port_deallocate(C.mach_task_self_, r.task)
		r.task = C.MACH_PORT_NULL
	}
	return nil
}

func (r *darwinMemReader) GetBinaryLoader() bin.BinaryLoader {
	return r.bin
}
