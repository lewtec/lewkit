package vulkan

import (
	"strings"
)

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum vendor.go

// Vendor is a PCI / Khronos vendor ID. String is the CLI / driver-id token.
// A value outside the set is unknown.
//
//lewkit:enum invalid=unknown
type Vendor uint32

const (
	VendorUnknown  Vendor = 0
	VendorAMD      Vendor = 0x1002
	VendorNVIDIA   Vendor = 0x10de
	VendorIntel    Vendor = 0x8086
	VendorARM      Vendor = 0x13b5
	VendorQualcomm Vendor = 0x5143
	VendorApple    Vendor = 0x106b
	VendorMesa     Vendor = 0x10005
)

// Uint32 is the PCI / Khronos vendor ID.
func (v Vendor) Uint32() uint32 { return uint32(v) }

// VendorFrom maps a PCI / Khronos vendor ID, falling back to the device name.
func VendorFrom(vendorID uint32, name string) Vendor {
	switch Vendor(vendorID) {
	case VendorAMD, VendorNVIDIA, VendorIntel, VendorARM, VendorQualcomm, VendorApple, VendorMesa:
		return Vendor(vendorID)
	}
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "nvidia"):
		return VendorNVIDIA
	case strings.Contains(lower, "amd") || strings.Contains(lower, "radeon") || strings.Contains(lower, "radv"):
		return VendorAMD
	case strings.Contains(lower, "intel"):
		return VendorIntel
	case strings.Contains(lower, "apple") || strings.Contains(lower, "moltenvk"):
		return VendorApple
	case strings.Contains(lower, "llvmpipe") || strings.Contains(lower, "mesa"):
		return VendorMesa
	}
	return VendorUnknown
}
