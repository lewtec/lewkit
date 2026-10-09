//go:build linux

package gtk

import "testing"

func TestGTK4StaysOutOfWebKit41Process(t *testing.T) {
	if !blockGTK4Choice(false, false, true) {
		t.Fatal("webkit 4.1 alone should keep GTK 4 out")
	}
	if blockGTK4Choice(false, true, true) {
		t.Fatal("webkit 6 should allow GTK 4")
	}
	if blockGTK4Choice(false, false, false) {
		t.Fatal("no webkit should allow GTK 4 dialogs")
	}
	if !blockGTK4Choice(true, true, false) {
		t.Fatal("a loaded GTK 3 blocks GTK 4")
	}
}

func TestPollBeforeInit(t *testing.T) {
	if Poll() != 0 {
		t.Fatal("poll before Ensure dispatched work")
	}
}

func TestAvailableOrSkip(t *testing.T) {
	if err := Available(t.Context()); err != nil {
		t.Skip(err)
	}
}
