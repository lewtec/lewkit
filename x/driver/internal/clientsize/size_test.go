package clientsize

import (
	"errors"
	"testing"
)

func TestDefault(t *testing.T) {
	invalid := errors.New("invalid size")
	w, h, err := Default(0, 0, invalid)
	if err != nil || w != 640 || h != 480 {
		t.Fatalf("%d %d %v", w, h, err)
	}
	w, h, err = Default(10, 20, invalid)
	if err != nil || w != 10 || h != 20 {
		t.Fatalf("%d %d %v", w, h, err)
	}
	_, _, err = Default(-1, 1, invalid)
	if !errors.Is(err, invalid) {
		t.Fatal(err)
	}
}
