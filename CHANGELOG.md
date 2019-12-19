## 0.4.0
- Enhance summary output 
- Show related channel info when goroutine is sending/receiving/select chan
- Fix: waitreason in 1.13("GC scavenge wait" added)
- Fix: summary height on top ui
- Fix: avoid crash on <1.12, since flushGen is added on 1.12
- Refactor: use reflect to do Binary.Initialize

## 0.3.0

- Refactor to make struct tags support more runtime types
- Add `gospy var` command to dump a runtime global variable
- Add `gospy heap` command to dump runtime heap info
- Dump target process's uptime
- Dump target process's last gc time
- Fix GC: TotalPauseTime

## 0.2.0

- Add e2e test, on different golang version
- Dump runtime info of scheduler
- Dump runtime info of memory and gc

## 0.1.0

initial release, support commands:

- gospy summary
- gospy top
