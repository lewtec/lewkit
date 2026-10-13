package tamago

import "testing"

func TestLinkFlags(t *testing.T) {
	want := []string{"-E", "cpuinit", "-T", "0x10010000", "-R", "0x1000"}
	if len(LinkFlags) != len(want) {
		t.Fatalf("flags %v", LinkFlags)
	}
	for i := range want {
		if LinkFlags[i] != want[i] {
			t.Fatalf("flags %v", LinkFlags)
		}
	}
	if RamStart+0x10000 != 0x10010000 {
		t.Fatalf("text address %x", RamStart+0x10000)
	}
}
