// Package vulkan exposes compute GPUs as drivers.
//
//	gpu, err := vulkan.Open(ctx)
//	defer gpu.Close()
//
// Import [github.com/lewtec/lewkit/x/driver/prelude] or this package
// so the factory registers. [List] returns every compute-capable GPU;
// [Open] takes the highest-weight handle (llvmpipe is weight 0).
// Libvulkan stays in [github.com/lewtec/lewkit/x/ffi/native/vulkan].
package vulkan

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	ffivulkan "github.com/lewtec/lewkit/x/ffi/native/vulkan"
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

// Buffer, Shader, Cmd, and ShaderConfig are the binding types programs use.
type (
	Buffer       = ffivulkan.Buffer
	Shader       = ffivulkan.Shader
	Cmd          = ffivulkan.Cmd
	ShaderConfig = ffivulkan.ShaderConfig
)

// Device is one compute-capable Vulkan GPU.
// The libvulkan handle stays inside this package.
type Device interface {
	Name() string
	Vendor() Vendor
	Type() DeviceType
	Close() error
	Buffer(size int) (*Buffer, error)
	Compile(ctx context.Context, cfg ShaderConfig) (*Shader, error)
	Begin() (*Cmd, error)
}

type device struct {
	binding *ffivulkan.Device
}

func (d *device) Name() string {
	if d == nil || d.binding == nil {
		return ""
	}
	return d.binding.Name()
}

func (d *device) Vendor() Vendor {
	if d == nil || d.binding == nil {
		return VendorUnknown
	}
	return d.binding.Vendor()
}

func (d *device) Type() DeviceType {
	if d == nil || d.binding == nil {
		return DeviceTypeOther
	}
	return d.binding.Type()
}

func (d *device) Buffer(size int) (*Buffer, error) {
	if d == nil || d.binding == nil {
		return nil, ffivulkan.ErrClosed
	}
	return d.binding.Buffer(size)
}

func (d *device) Compile(ctx context.Context, cfg ShaderConfig) (*Shader, error) {
	if d == nil || d.binding == nil {
		return nil, ffivulkan.ErrClosed
	}
	return d.binding.Compile(ctx, cfg)
}

func (d *device) Begin() (*Cmd, error) {
	if d == nil || d.binding == nil {
		return nil, ffivulkan.ErrClosed
	}
	return d.binding.Begin()
}

func (d *device) Close() error {
	if d == nil || d.binding == nil {
		return nil
	}
	err := d.binding.Close()
	d.binding = nil
	return err
}

func wrap(binding *ffivulkan.Device) Device {
	if binding == nil {
		return nil
	}
	return &device{binding: binding}
}

// Open returns the highest-weight compatible GPU.
func Open(ctx context.Context) (Device, error) {
	return driver.Get[Device](ctx)
}

// List returns every compatible GPU without constructing them.
func List(ctx context.Context) ([]driver.Handle[Device], error) {
	return driver.List[Device](ctx)
}
