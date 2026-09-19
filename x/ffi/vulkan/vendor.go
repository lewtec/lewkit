package vulkan

import (
	"fmt"
	"strings"
)

// Vendor is a PCI / Khronos vendor ID. String is the CLI / driver-id token.
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

// ParseVendor maps a String() token such as amd or nvidia.
func ParseVendor(name string) (Vendor, error) {
	var zero Vendor
	for _, known := range zero.Values() {
		if known.String() == name {
			return known, nil
		}
	}
	return VendorUnknown, fmt.Errorf("%w: %q", ErrVendor, name)
}

func (v Vendor) String() string {
	switch v {
	case VendorAMD:
		return "amd"
	case VendorNVIDIA:
		return "nvidia"
	case VendorIntel:
		return "intel"
	case VendorARM:
		return "arm"
	case VendorQualcomm:
		return "qualcomm"
	case VendorApple:
		return "apple"
	case VendorMesa:
		return "mesa"
	default:
		return "unknown"
	}
}

// Values lists Vendor members for [github.com/lewtec/lewkit/x/cmd.EnumArg].
func (Vendor) Values() []Vendor {
	return []Vendor{VendorUnknown, VendorAMD, VendorNVIDIA, VendorIntel, VendorARM, VendorQualcomm, VendorApple, VendorMesa}
}

func vendorFrom(vendorID uint32, name string) Vendor {
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
