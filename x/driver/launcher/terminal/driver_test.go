package terminal

import (
	"os"
	"testing"
)

func TestDisabledWhenStdioIsNotATTY(t *testing.T) {
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("TERMUX_VERSION", "")
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		read.Close()
		write.Close()
	})
	oldOut := os.Stdout
	os.Stdout = write
	t.Cleanup(func() { os.Stdout = oldOut })
	if err := (base{}).CheckCompatibility(t.Context()); err == nil {
		t.Fatal("terminal launcher is compatible when stdout is a pipe")
	}
}
