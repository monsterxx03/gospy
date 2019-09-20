# gospy: Non-Invasive goroutine inspector

A tiny tool to inspect your go process's goroutine info, without single line change in your code. Inspired by [py-spy](https://github.com/benfred/py-spy),
learned a lot from [delve](https://github.com/go-delve/delve)


## Usage

Get summary info about go process, and what every goroutine is doing:  `sudo ./gospy summary  --pid 1234`

    bin: /home/will/Downloads/prometheus-2.12.0.linux-amd64/prometheus, goVer: 1.12.8, gomaxprocs: 6
    P0 idle, schedtick: 642, syscalltick: 81
    P1 idle, schedtick: 959, syscalltick: 67
    P2 idle, schedtick: 992, syscalltick: 32
    P3 idle, schedtick: 581, syscalltick: 17
    P4 idle, schedtick: 89, syscalltick: 8
    P5 idle, schedtick: 231, syscalltick: 5
    Threads: 14 total, 0 running, 14 sleeping, 0 stopped, 0 zombie
    Goroutines: 44 total, 0 idle, 0 running, 5 syscall, 39 waiting

    goroutines:

    1 - waiting for chan receive: rt0_go (/usr/local/go/src/runtime/asm_amd64.s:202) 
    2 - waiting for force gc (idle): 5 (/usr/local/go/src/runtime/proc.go:240) 
    3 - waiting for GC sweep wait: gcenable (/usr/local/go/src/runtime/mgc.go:209) 
    8 - syscall: addtimerLocked (/usr/local/go/src/runtime/time.go:169) 
    9 - waiting for select: 0 (/app/vendor/go.opencensus.io/stats/view/worker.go:33) 
    16 - waiting for GC worker (idle): gcBgMarkStartWorkers (/usr/local/go/src/runtime/mgc.go:1785) 
    17 - waiting for finalizer wait: createfing (/usr/local/go/src/runtime/mfinal.go:156) 
    19 - syscall: 0 (/usr/local/go/src/os/signal/signal_unix.go:30) 
    22 - waiting for GC worker (idle): gcBgMarkStartWorkers (/usr/local/go/src/runtime/mgc.go:1785) 
    23 - waiting for GC worker (idle): gcBgMarkStartWorkers (/usr/local/go/src/runtime/mgc.go:1785) 
    38 - waiting for GC worker (idle): gcBgMarkStartWorkers (/usr/local/go/src/runtime/mgc.go:1785) 
    49 - waiting for GC worker (idle): gcBgMarkStartWorkers (/usr/local/go/src/runtime/mgc.go:1785) 
    50 - waiting for GC worker (idle): gcBgMarkStartWorkers (/usr/local/go/src/runtime/mgc.go:1785) 
    74 - waiting for select: sync (/app/scrape/scrape.go:408) 
    75 - syscall: addtimerLocked (/usr/local/go/src/runtime/time.go:169) 
    84 - syscall: addtimerLocked (/usr/local/go/src/runtime/time.go:169) 
    85 - waiting for select: Run (/app/vendor/github.com/oklog/run/group.go:36) 
    ...


A top like interface, aggregate goroutines by functions: `sudo ./gospy top --pid 1234`


![top](images/top.png)


## Limitation

- x86_64 linux only
- Don't work with binaries without debug info, if you build with linker flag `-w -s`, gospy won't be able to figure out function/variable names. 
- Don't work with binaries build with pie mode(go build -buildmode=pie), eg: official released dockerd binaries. For some reason, pie binary keeps debug info, but stripped
 the `.gopclntab` section, don't know how to handle it so far...


## TODO

- Optimize performance
- Support inspect go process run in container namespace
