package proc

import "debug/gosym"

// Constants for goroutine status (must match runtime2.go exactly)
const (
	_Gidle            = iota // 0
	_Grunnable               // 1
	_Grunning                // 2
	_Gsyscall                // 3
	_Gwaiting                // 4
	_Gmoribund_unused        // 5
	_Gdead                   // 6
	_Genqueue_unused         // 7
	_Gcopystack              // 8
	_Gpreempted              // 9

	// _Gscan bit is OR'd with state when GC is scanning the stack
	_Gscan          = 0x1000
	_Gscanrunnable  = _Gscan + _Grunnable  // 0x1001
	_Gscanrunning   = _Gscan + _Grunning   // 0x1002
	_Gscansyscall   = _Gscan + _Gsyscall   // 0x1003
	_Gscanwaiting   = _Gscan + _Gwaiting   // 0x1004
	_Gscanpreempted = _Gscan + _Gpreempted // 0x1009

	maxStackDepth = 100
)

// Status strings that match runtime's representation
var gStatusMap = map[uint32]string{
	_Gidle:            "idle",
	_Grunnable:        "runnable",
	_Grunning:         "running",
	_Gsyscall:         "syscall",
	_Gwaiting:         "waiting",
	_Gmoribund_unused: "moribund", // unused but present in runtime
	_Gdead:            "dead",
	_Genqueue_unused:  "enqueue", // unused but present in runtime
	_Gcopystack:       "copystack",
	_Gpreempted:       "preempted",

	// Scan states
	_Gscanrunnable:  "scanrunnable",
	_Gscanrunning:   "scanrunning",
	_Gscansyscall:   "scansyscall",
	_Gscanwaiting:   "scanwaiting",
	_Gscanpreempted: "scanpreempted",
}

type G struct {
	Address       uint64 `json:"address"`         // goroutine structure address
	Goid          int64  `json:"go_id"`           // goroutine ID
	Status        string `json:"status"`          // goroutine status
	WaitReason    string `json:"wait_reason"`     // wait reason
	Stack         Stack  `json:"stack"`           // Stack info
	M             uint64 `json:"-"`               // associated M structure address
	Sched         Sched  `json:"sched"`           // scheduling info
	AtomicStatus  uint32 `json:"-"`               // raw status value
	FuncName      string `json:"func_name"`       // currently running function name
	StartPC       uint64 `json:"start_pc"`        // starting function address
	StartFuncName string `json:"start_func_name"` // starting function name
}

type Stack struct {
	Lo uint64 `json:"lo"`
	Hi uint64 `json:"hi"`
}

type Sched struct {
	PC uint64 `json:"pc"` // program counter
	SP uint64 `json:"sp"` // stack pointer
}

type StackFrame struct {
	PC       uint64
	SP       uint64
	Function string
	File     string
	Line     int
	Func     *gosym.Func // Store the gosym Func for potential later use
}
