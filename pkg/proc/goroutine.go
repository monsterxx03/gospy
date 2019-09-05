package proc

// G is goroutine struct parsed from process memory and binary dwarf
type G struct {
	ID     int    "goid"
	Gopc   uint64 "gopc"
	Pc     uint64 "sched.pc"
	Status uint64 "atomicstatus"
}
