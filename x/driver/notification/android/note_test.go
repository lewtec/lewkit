package android

import "testing"

func TestPercentOf(t *testing.T) {
	if percentOf(0.5, false) != 0 {
		t.Fatal(percentOf(0.5, false))
	}
	if percentOf(0.5, true) != 50 {
		t.Fatal(percentOf(0.5, true))
	}
	if percentOf(1, true) != 100 || percentOf(-1, true) != 0 || percentOf(2, true) != 100 {
		t.Fatal(percentOf(1, true), percentOf(-1, true), percentOf(2, true))
	}
}
