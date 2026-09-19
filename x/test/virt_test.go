package test

import "testing"

func TestVirtSize(t *testing.T) {
	n := VirtSize(t)
	if n <= 0 {
		t.Fatalf("VirtSize = %d", n)
	}
}
