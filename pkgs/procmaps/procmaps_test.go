package procmaps

import (
	"testing"
)

func TestReadProcMaps(t *testing.T) {
	ranges, err := parseProcMaps("testdata/maps")
	if err != nil {
		t.Error(err)
	}
	testsData := []Range{
		Range{Start: 0x400000, End: 0x005af000,
			Perm: "r-xp", Offset: 0x00000000,
			Dev: "103:02", Inode: 5789181, Filename: "/bin/snet"},
		Range{Start: 0x7fa392342000, End: 0x7fa392343000,
			Perm: "rw-p", Offset: 0x28000, Dev: "103:02", Inode: 12980870, Filename: "/lib/x86_64-linux-gnu/ld-2.27.so"},
		Range{Start: 0x7fa392343000, End: 0x7fa392344000, Perm: "rw-p",
			Offset: 0x00000000, Dev: "00:00", Inode: 0, Filename: ""},
		Range{Start: 0xffffffffff600000, End: 0xffffffffff601000, Perm: "r-xp",
			Offset: 0x00000000, Dev: "00:00", Inode: 0, Filename: "[vsyscall]"},
	}
	for i, rng := range ranges {
		if rng != testsData[i] {
			t.Errorf("Not equal: %v, %v", rng, testsData[i])
		}
	}
}
