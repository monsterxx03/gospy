package proc

import (
	"time"
	_ "unsafe" // required to use //go:linkname
)

// G represents a goroutine with detailed information
type Runtime struct {
	InitTime  int64  `json:"-"`          // when runtime was initialized(monotime)
	GoVersion string `json:"go_version"` // Go runtime version
}

func (r Runtime) Uptime() time.Duration {
	return time.Duration(nanotime() - r.InitTime)
}

type runtimeCache struct {
	runtime *Runtime
}

var runtimeInfoCache = make(map[int]*runtimeCache) // pid -> runtime info

func (r *commonMemReader) RuntimeInfo() (*Runtime, error) {
	// Check cache first
	if cached, ok := runtimeInfoCache[r.pid]; ok {
		return cached.runtime, nil
	}

	rt := &Runtime{}

	// Parse Go version
	goVersionAddr, err := r.GetBinaryLoader().FindVariableAddress("runtime.buildVersion")
	if err == nil {
		version, err := r.readString(r.GetStaticBase() + goVersionAddr)
		if err == nil {
			rt.GoVersion = version
		}
	}

	// Parse runtime init time (monotime)
	initTimeAddr, err := r.GetBinaryLoader().FindVariableAddress("runtime.runtimeInitTime")
	if err == nil {
		initTimeNano, err := r.readUint64(r.GetStaticBase() + initTimeAddr)
		if err == nil {
			rt.InitTime = int64(initTimeNano)
		}
	}

	// Cache the result
	runtimeInfoCache[r.pid] = &runtimeCache{runtime: rt}
	return rt, nil
}

//go:noescape
//go:linkname nanotime runtime.nanotime
func nanotime() int64
