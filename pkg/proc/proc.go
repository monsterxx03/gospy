package proc

// NewProcessMemReader creates a new memory reader for the specified process.
// On Linux it uses /proc/<pid>/mem, on Darwin it uses mach_vm_read.
func NewProcessMemReader(pid int, binPath string) (ProcessMemReader, error) {
	return newProcessMemReader(pid, binPath)
}
