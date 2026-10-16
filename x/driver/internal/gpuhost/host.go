// Package gpuhost is the host check shared by a present backend and its evaluator.
// OpenGL stays split: present calls Available and the evaluator calls ComputeAvailable.
package gpuhost

import (
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	ffid3d12 "github.com/lewtec/lewkit/x/ffi/native/d3d12"
	ffimetal "github.com/lewtec/lewkit/x/ffi/native/metal"
)

// Metal reports whether this process can open a Metal device.
// Anything other than darwin or ios is incompatible.
func Metal() error {
	if runtime.GOOS != "darwin" && runtime.GOOS != "ios" {
		return fmt.Errorf("%w: not apple", driver.ErrIncompatible)
	}
	if err := ffimetal.Available(); err != nil {
		return driver.WrapIncompatible(err)
	}
	return nil
}

// Direct3D reports whether this process can open a Direct3D 12 device.
// The host must be windows on amd64 or arm64.
func Direct3D() error {
	if err := driver.ForGOOS("windows"); err != nil {
		return err
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
	}
	if err := ffid3d12.Available(); err != nil {
		return driver.WrapIncompatible(err)
	}
	return nil
}
