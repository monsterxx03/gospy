package proc

// G is goroutine struct parsed from process memory and binary dwarf
type G struct {
	id   int    "goid"
	gopc uint64 "gopc"
	pc   uint64 "sched.pc"
}
