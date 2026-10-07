//go:build android

package android

import "testing"

func TestNotedVMSkipsLoad(t *testing.T) {
	resetNote()
	t.Cleanup(resetNote)
	NoteVM(1)
	NoteLoader()
	n, err := JavaVMs()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("vms %d", n)
	}
	on, err := OnLooper()
	if err != nil {
		t.Fatal(err)
	}
	if !on {
		t.Fatal("loader thread is not the looper")
	}
}
