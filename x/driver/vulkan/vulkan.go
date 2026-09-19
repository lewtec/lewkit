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

// DeviceType is [ffivulkan.DeviceType]: software, integrated, dedicated, virtual.
type DeviceType = ffivulkan.DeviceType

const (
	DeviceTypeOther      = ffivulkan.DeviceTypeOther
	DeviceTypeIntegrated = ffivulkan.DeviceTypeIntegrated
	DeviceTypeDedicated  = ffivulkan.DeviceTypeDedicated
	DeviceTypeVirtual    = ffivulkan.DeviceTypeVirtual
	DeviceTypeSoftware   = ffivulkan.DeviceTypeSoftware
)

// ParseDeviceType maps a String() token such as dedicated or software.
func ParseDeviceType(name string) (DeviceType, error) {
	return ffivulkan.ParseDeviceType(name)
}

// Vendor is [ffivulkan.Vendor]: amd, nvidia, intel, …
type Vendor = ffivulkan.Vendor

const (
	VendorUnknown  = ffivulkan.VendorUnknown
	VendorAMD      = ffivulkan.VendorAMD
	VendorNVIDIA   = ffivulkan.VendorNVIDIA
	VendorIntel    = ffivulkan.VendorIntel
	VendorARM      = ffivulkan.VendorARM
	VendorQualcomm = ffivulkan.VendorQualcomm
	VendorApple    = ffivulkan.VendorApple
	VendorMesa     = ffivulkan.VendorMesa
)

// VendorFrom maps a PCI / Khronos vendor ID, falling back to the device name.
func VendorFrom(vendorID uint32, name string) Vendor {
	return ffivulkan.VendorFrom(vendorID, name)
}

// Device is one compute-capable Vulkan GPU.
type Device interface {
	Name() string
	Vendor() Vendor
	Type() DeviceType
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

func (d *device) Vendor() Vendor {
	if d == nil || d.native == nil {
		return VendorUnknown
	}
	return d.native.Vendor()
}

func (d *device) Type() DeviceType {
	if d == nil || d.native == nil {
		return DeviceTypeOther
	}
	return d.native.Type()
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
