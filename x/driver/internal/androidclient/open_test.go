package androidclient

import (
	"errors"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
)

func TestOpenNotAndroid(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip()
	}
	_, err := Open(t.Context())
	if !errors.Is(err, driver.ErrIncompatible) {
		t.Fatal(err)
	}
}

func TestInteractiveNotAndroid(t *testing.T) {
	if runtime.GOOS == "android" {
		t.Skip()
	}
	err := Interactive(t.Context())
	if !errors.Is(err, driver.ErrIncompatible) {
		t.Fatal(err)
	}
}
