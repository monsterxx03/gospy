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
