# gospy: Non-Invasive goroutine inspector

A tiny tool to inspect your go process's gorouting info, without single line change in your code. Inspired by [py-spy](https://github.com/benfred/py-spy),
learned a lot from [delve](https://github.com/go-delve/delve)


## Usage

Get summary info about go process, and what every goroutine is doing:  `sudo ./gospy summary  --pid 1234`

    bin: /home/will/repos/snet/bin/snet, goVer: 1.12.1
    Threads: 133 total, 0 running, 133 sleeping, 0 stopped, 0 zombie
    Goroutines: 242 total, 0 idle, 0 running, 112 syscall, 130 waiting

    goroutines:

    1 - waiting for chan receive: rt0_go (/usr/local/go/src/runtime/asm_amd64.s:202) 
    2 - waiting for force gc (idle): 5 (/usr/local/go/src/runtime/proc.go:240) 
    3 - waiting for GC sweep wait: gcenable (/usr/local/go/src/runtime/mgc.go:209) 
    4 - waiting for timer goroutine (idle): addtimerLocked (/usr/local/go/src/runtime/time.go:169) 
    7 - waiting for GC worker (idle): gcBgMarkStartWorkers (/usr/local/go/src/runtime/mgc.go:1785) 
    8 - waiting for GC worker (idle): gcBgMarkStartWorkers (/usr/local/go/src/runtime/mgc.go:1785) 
    9 - waiting for GC worker (idle): gcBgMarkStartWorkers (/usr/local/go/src/runtime/mgc.go:1785) 
    12 - waiting for IO wait: Run (/home/will/repos/snet/server.go:46) 
    13 - syscall: handle (/home/will/repos/snet/server.go:71) 
    15 - waiting for semacquire: Run (/home/will/repos/snet/server.go:46) 
    16 - syscall: handle (/home/will/repos/snet/server.go:71) 
    18 - waiting for finalizer wait: createfing (/usr/local/go/src/runtime/mfinal.go:156) 
    19 - waiting for sleep: newReqList (/home/will/go/pkg/mod/github.com/shadowsocks/shadowsocks-go@v0.0.0-20190307081127-ac922d10041c/shadowsocks/udprelay.go:86) 
    20 - syscall: 0 (/usr/local/go/src/os/signal/signal_unix.go:30) 
    27 - waiting for IO wait: main (/home/will/repos/snet/main.go:95) 


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
