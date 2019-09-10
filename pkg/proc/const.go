package proc

// amd64 pointer size
const POINTER_SIZE = 8

// from runtime/runtime2.go
const (
	gidle            = iota // 0
	grunnable               // 1
	grunning                // 2
	gsyscall                // 3
	gwaiting                // 4
	gmoribund_unused        // 5
	gdead                   // 6
	genqueue_unused         // 7
	gcopystack              // 8
	gscan            = 0x1000
	gscanrunnable    = gscan + grunnable // 0x1001
	gscanrunning     = gscan + grunning  // 0x1002
	gscansyscall     = gscan + gsyscall  // 0x1003
	gscanwaiting     = gscan + gwaiting  // 0x1004
)

var gstatusStrings = [...]string{
	gidle:            "idle",
	grunnable:        "runnable",
	grunning:         "running",
	gsyscall:         "syscall",
	gwaiting:         "waiting",
	gmoribund_unused: "moribund_unused",
	gdead:            "dead",
	genqueue_unused:  "enqueue_unused",
	gcopystack:       "copystack",
	gscan:            "scan",
	gscanrunnable:    "scanrunnable",
	gscanrunning:     "scanrunning",
	gscansyscall:     "scansyscall",
	gscanwaiting:     "scanwaiting",
}

// from runtime/runtime2.go
const (
	waitReasonZero                  gwaitReason = iota // ""
	waitReasonGCAssistMarking                          // "GC assist marking"
	waitReasonIOWait                                   // "IO wait"
	waitReasonChanReceiveNilChan                       // "chan receive (nil chan)"
	waitReasonChanSendNilChan                          // "chan send (nil chan)"
	waitReasonDumpingHeap                              // "dumping heap"
	waitReasonGarbageCollection                        // "garbage collection"
	waitReasonGarbageCollectionScan                    // "garbage collection scan"
	waitReasonPanicWait                                // "panicwait"
	waitReasonSelect                                   // "select"
	waitReasonSelectNoCases                            // "select (no cases)"
	waitReasonGCAssistWait                             // "GC assist wait"
	waitReasonGCSweepWait                              // "GC sweep wait"
	waitReasonChanReceive                              // "chan receive"
	waitReasonChanSend                                 // "chan send"
	waitReasonFinalizerWait                            // "finalizer wait"
	waitReasonForceGGIdle                              // "force gc (idle)"
	waitReasonSemacquire                               // "semacquire"
	waitReasonSleep                                    // "sleep"
	waitReasonSyncCondWait                             // "sync.Cond.Wait"
	waitReasonTimerGoroutineIdle                       // "timer goroutine (idle)"
	waitReasonTraceReaderBlocked                       // "trace reader (blocked)"
	waitReasonWaitForGCCycle                           // "wait for GC cycle"
	waitReasonGCWorkerIdle                             // "GC worker (idle)"
)

var gwaitReasonStrings = [...]string{
	waitReasonZero:                  "",
	waitReasonGCAssistMarking:       "GC assist marking",
	waitReasonIOWait:                "IO wait",
	waitReasonChanReceiveNilChan:    "chan receive (nil chan)",
	waitReasonChanSendNilChan:       "chan send (nil chan)",
	waitReasonDumpingHeap:           "dumping heap",
	waitReasonGarbageCollection:     "garbage collection",
	waitReasonGarbageCollectionScan: "garbage collection scan",
	waitReasonPanicWait:             "panicwait",
	waitReasonSelect:                "select",
	waitReasonSelectNoCases:         "select (no cases)",
	waitReasonGCAssistWait:          "GC assist wait",
	waitReasonGCSweepWait:           "GC sweep wait",
	waitReasonChanReceive:           "chan receive",
	waitReasonChanSend:              "chan send",
	waitReasonFinalizerWait:         "finalizer wait",
	waitReasonForceGGIdle:           "force gc (idle)",
	waitReasonSemacquire:            "semacquire",
	waitReasonSleep:                 "sleep",
	waitReasonSyncCondWait:          "sync.Cond.Wait",
	waitReasonTimerGoroutineIdle:    "timer goroutine (idle)",
	waitReasonTraceReaderBlocked:    "trace reader (blocked)",
	waitReasonWaitForGCCycle:        "wait for GC cycle",
	waitReasonGCWorkerIdle:          "GC worker (idle)",
}

// from: man proc
var threadStateStrings = map[string]string{
	"R": "Running",
	"S": "Sleeping",
	"D": "Disk sleep",
	"Z": "Zombie",
	"T": "Stopped",
	"t": "Tracing stop",
	"w": "Paging",
	"x": "Dead",
	"X": "Dead",
	"K": "Wakekill",
	"W": "Waking",
	"P": "Parked",
}
