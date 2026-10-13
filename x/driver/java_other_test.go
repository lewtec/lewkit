//go:build !android

package driver

import (
	"errors"
	"testing"
)

func TestJavaVM(t *testing.T) {
	err := JavaVM()
	if !errors.Is(err, ErrIncompatible) || err.Error() != "component is incompatible: no Java VM" {
		t.Fatal(err)
	}
}
