## 0.3.0(next)

- refactor to make struct tags support more runtime types
- Add `go var` command to dump a runtime global variable
- Fix GC: TotalPauseTime
- Dump target process's uptime
- Dump target process's last gc time

## 0.2.0

- Add e2e test, on different golang version
- Dump runtime info of scheduler
- Dump runtime info of memory and gc

## 0.1.0

initial release, support commands:

- gospy summary
- gospy top
