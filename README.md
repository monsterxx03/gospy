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

    1 - waiting for chan receive: main (/usr/local/go/src/runtime/proc.go:110)
    2 - waiting for force gc (idle): forcegchelper (/usr/local/go/src/runtime/proc.go:242)
    3 - waiting for GC sweep wait: bgsweep (/usr/local/go/src/runtime/mgcsweep.go:64)
    7(M8)- syscall: timerproc (/usr/local/go/src/runtime/time.go:247)
    8 - waiting for select: start (/app/vendor/go.opencensus.io/stats/view/worker.go:149)
    14 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    15 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    17 - waiting for finalizer wait: runfinq (/usr/local/go/src/runtime/mfinal.go:161)
    19(M4)- syscall: loop (/usr/local/go/src/os/signal/signal_unix.go:21)
    26 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    27 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    28 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    36 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    49 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    50 - waiting for GC worker (idle): gcBgMarkWorker (/usr/local/go/src/runtime/mgc.go:1807)
    51(M7)- syscall: timerproc (/usr/local/go/src/runtime/time.go:247)
    52 - waiting for select: func1 (/app/vendor/github.com/oklog/run/group.go:37)
    53 - waiting for chan receive: func1 (/app/vendor/github.com/oklog/run/group.go:37)
    54 - waiting for chan receive: func1 (/app/vendor/github.com/oklog/run/group.go:37)
    ...


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

## Target process go version compatibility

Works on target go version >= 1.10, the DWARF info in binary is different before 1.10: https://golang.org/doc/go1.10#compiler.
Current code won't work.

tested:

- [x] 1.10
- [x] 1.10.8
- [x] 1.11.13
- [x] 1.12.9 
- [x] 1.13.1

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

- Dump heap info
- Support dump more variable types
- Optimize performance
