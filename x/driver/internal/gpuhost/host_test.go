package gpuhost

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
)

func TestMetalHost(t *testing.T) {
	err := Metal()
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		return
	}
	if !errors.Is(err, driver.ErrIncompatible) || !strings.Contains(err.Error(), "not apple") {
		t.Fatal(err)
	}
}

func TestDirect3DHost(t *testing.T) {
	err := Direct3D()
	if runtime.GOOS == "windows" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") {
		return
	}
	if !errors.Is(err, driver.ErrIncompatible) || !strings.Contains(err.Error(), "not windows") {
		t.Fatal(err)
	}
}
