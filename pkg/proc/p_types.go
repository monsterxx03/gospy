package proc

import "fmt"

// Constants for P status (must match runtime2.go exactly)
const (
	_Pidle = iota
	_Prunning
	_Psyscall
	_Pgcstop
	_Pdead
)

type P struct {
	Address   uint64 `json:"address"`    // P structure address
	ID        int32  `json:"id"`         // P ID
	Status    string `json:"status"`     // P status
	MCache    uint64 `json:"m_cache"`    // Per-P cache for small objects
	SchedTick uint32 `json:"sched_tick"` // Tick counter for scheduler
}

func parsePStatus(status uint32) string {
	switch status {
	case _Pidle:
		return "idle"
	case _Prunning:
		return "running"
	case _Psyscall:
		return "syscall"
	case _Pgcstop:
		return "gcstop"
	case _Pdead:
		return "dead"
	default:
		return fmt.Sprintf("unknown(%d)", status)
	}
}
