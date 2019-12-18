package proc

import "strings"

type gwaitReason uint8

// from runtime/runtime2.go
type waitReasonMapping map[gwaitReason]string
var waitReason_1_10 = waitReasonMapping{
	0: "",
	1:       "GC assist marking",
	2:                "IO wait",
	3:    "chan receive (nil chan)",
	4:       "chan send (nil chan)",
	5:           "dumping heap",
	6:     "garbage collection",
	7: "garbage collection scan",
	8:             "panicwait",
	9:                "select",
	10:         "select (no cases)",
	11:          "GC assist wait",
	12:           "GC sweep wait",
	13:           "chan receive",
	14:              "chan send",
	15:         "finalizer wait",
	16:           "force gc (idle)",
	17:            "semacquire",
	18:                 "sleep",
	19:          "sync.Cond.Wait",
	20:    "timer goroutine (idle)",
	21:    "trace reader (blocked)",
	22:        "wait for GC cycle",
	23:          "GC worker (idle)",
}

var waitReason_1_11 = waitReason_1_10
var waitReason_1_12 = waitReason_1_10
var waitReason_1_13 = waitReasonMapping{
	0: "",
	1:       "GC assist marking",
	2:                "IO wait",
	3:    "chan receive (nil chan)",
	4:       "chan send (nil chan)",
	5:           "dumping heap",
	6:     "garbage collection",
	7: "garbage collection scan",
	8:             "panicwait",
	9:                "select",
	10:         "select (no cases)",
	11:          "GC assist wait",
	12:           "GC sweep wait",
	13:        "GC scavenge wait",  // added in 1.13, since it's added in middle, we must duplicate this list for different versions
	14:           "chan receive",
	15:              "chan send",
	16:         "finalizer wait",
	17:           "force gc (idle)",
	18:            "semacquire",
	19:                 "sleep",
	20:          "sync.Cond.Wait",
	21:    "timer goroutine (idle)",
	22:    "trace reader (blocked)",
	23:        "wait for GC cycle",
	24:          "GC worker (idle)",
	25: "preempted",   // added in 1.14
}

var waitReason_default = waitReason_1_13

var versionedWaitReason = map[string]waitReasonMapping{
	"1.10": waitReason_1_10,
	"1.11": waitReason_1_11,
	"1.12": waitReason_1_12,
	"1.13": waitReason_1_13,
}

func getWaitReasonMap(version string) waitReasonMapping{
	// take main version
	if strings.Count(version, ".") == 2 {
		splited := strings.Split(version, ".")
		version = splited[0] + "." + splited[1]
	}
	mapping, ok := versionedWaitReason[version]
	if ok {
		return mapping
	}
	return waitReason_default
}
