# gospy: Non-Invasive goroutine inspector

A tiny tool to inspect your go process's goroutine info, without single line change in your code. Inspired by [py-spy](https://github.com/benfred/py-spy),
learned a lot from [delve](https://github.com/go-delve/delve)


## Usage

Get summary info about go process, and what every goroutine is doing:  `sudo ./gospy summary  --pid 1234`

    bin: /home/will/Downloads/prometheus-2.12.0.linux-amd64/prometheus, goVer: 1.12.8, gomaxprocs: 6
    Sched: NMidle 6, NMspinning 0, NMfreed 0, NPidle 5, NGsys 16, Runqsize: 0
    P0 idle, schedtick: 642, syscalltick: 81, curM: M0
    P1 idle, schedtick: 959, syscalltick: 67, curM: nil
    P2 idle, schedtick: 992, syscalltick: 32, curM: nil
    P3 idle, schedtick: 581, syscalltick: 17, curM: nil
    P4 idle, schedtick: 89, syscalltick: 8, curM: nil
    P5 idle, schedtick: 231, syscalltick: 5, curM: nil
    Threads: 14 total, 0 running, 14 sleeping, 0 stopped, 0 zombie
    Goroutines: 44 total, 0 idle, 0 running, 5 syscall, 39 waiting

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



A top like interface, aggregate goroutines by functions: `sudo ./gospy top --pid 1234`


![top](images/top.png)


## Limitation

- x86_64 linux only
- Don't work with binaries without debug info, if you build with linker flag `-w -s`, gospy won't be able to figure out function/variable names. 
- Don't work with binaries build with pie mode(go build -buildmode=pie), eg: official released dockerd binaries. For some reason, pie binary keeps debug info, but stripped
 the `.gopclntab` section, don't know how to handle it so far...


## TODO

- [] Optimize performance
