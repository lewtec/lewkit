// Package vulkan exposes compute GPUs as drivers.
//
//	gpu, err := vulkan.Open(ctx)
//	defer gpu.Close()
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or this package
// so the factory registers. [List] returns every compute-capable GPU;
// [Open] takes the highest-weight handle (llvmpipe is weight 0).
package vulkan

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	ffivulkan "github.com/lewtec/lewkit/x/ffi/vulkan"
)

// Device is one compute-capable Vulkan GPU.
type Device interface {
	Name() string
	Vendor() string
	Native() *ffivulkan.Device
	Close() error
}

type device struct {
	native *ffivulkan.Device
}

func (d *device) Name() string {
	if d == nil || d.native == nil {
		return ""
	}
	return d.native.Name()
}

func (d *device) Vendor() string {
	if d == nil || d.native == nil {
		return ""
	}
	return d.native.Vendor()
}

func (d *device) Native() *ffivulkan.Device {
	if d == nil {
		return nil
	}
	return d.native
}

func (d *device) Close() error {
	if d == nil || d.native == nil {
		return nil
	}
	err := d.native.Close()
	d.native = nil
	return err
}

func wrap(native *ffivulkan.Device) Device {
	if native == nil {
		return nil
	}
	return &device{native: native}
}

// Open returns the highest-weight compatible GPU.
func Open(ctx context.Context) (Device, error) {
	return driver.Get[Device](ctx)
}

// List returns every compatible GPU without constructing them.
func List(ctx context.Context) ([]driver.Handle[Device], error) {
	return driver.List[Device](ctx)
}
