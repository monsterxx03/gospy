package proc

import "fmt"

type P struct {
	Address      uint64 `json:"address"`       // P structure address
	ID           int32  `json:"id"`            // P ID
	Status       string `json:"status"`        // P status
	StatusString string `json:"status_string"` // Human readable status
	MCache       uint64 `json:"m_cache"`       // Per-P cache for small objects
	SchedTick    uint32 `json:"sched_tick"`    // Tick counter for scheduler
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
