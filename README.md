![](https://github.com/monsterxx03/gospy/workflows/Go/badge.svg)

# gospy: Non-Invasive goroutine inspector

A tiny tool to inspect your go process's goroutine info, without single line change in your code. Inspired by [py-spy](https://github.com/benfred/py-spy),
learned a lot from [delve](https://github.com/go-delve/delve)


## Usage

### summary

`sudo ./gospy summary  --pid 1234`


    bin: /bin/prometheus, goVer: 1.12.8, gomaxprocs: 8, uptime: 2h20m23s
    Sched: NMidle 9, NMspinning 0, NMfreed 0, NPidle 8, NGsys 20, Runqsize: 0
    Heap: HeapInUse 16.70MB, HeapSys 62.78MB, HeapLive 14.49MB, HeapObjects 65076, Nmalloc 593803, Nfree 528731
    GC: TotalPauseTime 3.28412ms, NumGC 27, NumForcedGC 0, GCCpu 0.000013, LastGC: 38.245s ago
    P0 idle schedtick: 5828 syscalltick: 3308 curM: nil runqsize: 71
    P1 idle schedtick: 4241 syscalltick: 3441 curM: nil runqsize: 67
    P2 idle schedtick: 1938 syscalltick: 2449 curM: nil runqsize: 47
    P3 idle schedtick: 7220 syscalltick: 1668 curM: nil runqsize: 21
    P4 idle schedtick: 2439 syscalltick: 1322 curM: nil runqsize: 39
    P5 idle schedtick: 1682 syscalltick: 146  curM: nil runqsize: 4
    P6 idle schedtick: 4342 syscalltick: 50   curM: nil runqsize: 2
    P7 idle schedtick: 2327 syscalltick: 17   curM: nil runqsize: 2
    Threads: 0 total, 0 running, 0 sleeping, 0 stopped, 0 zombie
    Goroutines: 53 total, 0 idle, 0 running, 7 syscall, 46 waiting

    goroutines:


    1 - waiting for chan receive: main (/usr/local/go/src/runtime/proc.go:110) <- make(chan interface(16), 10)
    2 - waiting for force gc (idle): forcegchelper (/usr/local/go/src/runtime/proc.go:242)
    3 - waiting for GC sweep wait: bgsweep (/usr/local/go/src/runtime/mgcsweep.go:64)
    4(M3)- syscall: loop (/usr/local/go/src/os/signal/signal_unix.go:21)
    12 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    13 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    17 - waiting for finalizer wait: runfinq (/usr/local/go/src/runtime/mfinal.go:161)
    23 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    35(M6)- syscall: timerproc (/usr/local/go/src/runtime/time.go:247)
    36 - waiting for select: start (/app/vendor/go.opencensus.io/stats/view/worker.go:149) select <- make(chan struct(24), 1)
    41 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    42 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    43 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    90(M12)- syscall: timerproc (/usr/local/go/src/runtime/time.go:247)
    91 - waiting for select: sender (/app/discovery/manager.go:251) select <- make(chan struct(24), 1)
    92 - waiting for select: watcher (/app/vendor/google.golang.org/grpc/balancer_conn_wrappers.go:113) select <- make(chan ptr(8), 1)
    93 - waiting for chan receive: 1 (/app/prompb/rpc.pb.gw.go:91) <- make(chan struct(0), 0)
    94 - waiting for chan receive: func3 (/app/web/web.go:513) <- make(chan interface(16), 1024)
    95 - waiting for chan receive: func4 (/app/web/web.go:516) <- make(chan interface(16), 1024)


If you know something about golang's GMP scheduling model, following may be instresting:

- If a `P` is running, it wil have an associated `M` (curM).
- A `G` (goroutine) maybe have a associated `M`, even it's not running(the M won't be binded to any P).


###  top

 aggregate goroutines by functions: `sudo ./gospy top --pid 1234`


![top](images/top.png)

### var

dump a global variable value by name: `sudo ./gospy var --pid 1234 --name runtime.ncpu`

    type: int32, value: 6
    
 Support types:
 
- int(8, 16, 32, 64)
- uint(8, 16, 32, 64)
- bool
- string

### heap

dump runtime heap info: `sudo ./gospy heap --pid 1234`

    
    PagesInUse: 918, PagesSwept: 977
    Large Object(>32KB) Stats: AllocNum: 21, AllocRamSize: 4.52MB, FreeNum: 16, FreedRamSize: 3.38MB
    SweepDone: 1, Sweepers: 0, Sweepgen: 82
    P0, FlushGen:82:
            Tiny size object(<16B): AllocNum: 9, BytesUsage: 11/16
            Large size object freed(>32KB): FreeNum: 0, FreedRamSize: 0B
            Small size object(<32KB):
                    0B~8B: npages: 1, allocCount: 114
                    8B~16B: npages: 1, allocCount: 259
                    16B~32B: npages: 1, allocCount: 146
                    32B~48B: npages: 1, allocCount: 21
                    48B~64B: npages: 1, allocCount: 40
                    64B~80B: npages: 1, allocCount: 88
                    80B~96B: npages: 1, allocCount: 76
                    96B~112B: npages: 1, allocCount: 17
                    112B~128B: npages: 1, allocCount: 48
                    144B~160B: npages: 1, allocCount: 26
                    160B~176B: npages: 1, allocCount: 13
                    176B~192B: npages: 1, allocCount: 14
                    208B~224B: npages: 1, allocCount: 24
                    240B~256B: npages: 1, allocCount: 19
                    256B~288B: npages: 1, allocCount: 15
                    288B~320B: npages: 1, allocCount: 16
                    352B~384B: npages: 1, allocCount: 19
                    480B~512B: npages: 1, allocCount: 14
                    512B~576B: npages: 1, allocCount: 1
                    576B~640B: npages: 1, allocCount: 5
                    704B~768B: npages: 1, allocCount: 6
                    768B~896B: npages: 1, allocCount: 7
                    896B~1.00KB: npages: 1, allocCount: 2
                    1.00KB~1.12KB: npages: 1, allocCount: 4
                    1.12KB~1.25KB: npages: 1, allocCount: 3
                    1.25KB~1.38KB: npages: 2, allocCount: 8
                    2.00KB~2.25KB: npages: 2, allocCount: 5
                    4.75KB~5.25KB: npages: 2, allocCount: 3
    P1, FlushGen: 82:
        ...

## Target process go version compatibility

Works on target go version >= 1.10, the DWARF info in binary is different before 1.10: https://golang.org/doc/go1.10#compiler.
Current code won't work.

tested:

- [x] 1.10.X
- [x] 1.11.X
- [x] 1.12.X 
- [x] 1.13.X

## Build from source

Minium go version to build: 1.12.0

git clone https://github.com/monsterxx03/gospy.git

gospy is based on go module, please don't put repo under GOPATH, it won't work.

cd gospy && make

binary will be created under gospy/bin/

## Limitation

- x86_64 linux only
- Don't work with binaries without debug info, if you build with linker flag `-w -s`, gospy won't be able to figure out function/variable names. 
- Don't work with binaries build with pie mode(go build -buildmode=pie), eg: official released dockerd binaries. For some reason, pie binary keeps debug info, but stripped
 the `.gopclntab` section, don't know how to handle it so far...

## FAQ

#### How it works?

Read DWARF ino from ELF binary(embed by go compiler), parse some basic global variables'(runtime.allgs, runtime.allglen...) virtual memory address, then read target process's memory space to recreate runtime structs(runtime.g, runtime.p, runtime.m, runtime.sched...)

#### How to read remote process's memory space?

There're three ways:

- `PTRACE_PEEKDATA`, it can only read a long word(8 bytes) at a time. If we want to read a continuous memory space, need to call it multi times.
- `process_vm_readv`, available after kernel 3.2, can read a continuous block of process memory space, not exposed directly in go, maybe need cgo to call?
- Read `/proc/{pid}/mem` directly, it's the easiest way on linux. Also more efficient than a syscall in go. 

gospy takes the third way(`/proc/{pid}/mem`). Bad side is sudo privilege is required.


#### Is there any overhead on remote process?

Yes. By default, gospy use `PTRACE_ATTACH` to suspend target process to get a consistent memory view, after reading, `PTRACE_DEATCH` to resume target process.

If `--non-blocking` option is provided, gospy will do memory reading directly, won't suspend target process. If target process is creating/destorying goroutines actively, it may fail during reading memory.

#### If target process's binary is striped, any workaround without restarting target process?

You can compile a binary with debug info and specify the `--bin` option. Ensure compile with same code revision, same go version.

#### Can gospy spy itself?

Yes :)

## TODO

- Support dump more variable types
- Optimize performance
