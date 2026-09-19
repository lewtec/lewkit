package vulkan

import "strings"

const (
	physicalDeviceTypeCPU = 4

	vendorIDAMD      = 0x1002
	vendorIDNVIDIA   = 0x10de
	vendorIDIntel    = 0x8086
	vendorIDARM      = 0x13b5
	vendorIDQualcomm = 0x5143
	vendorIDApple    = 0x106b
	vendorIDMesa     = 0x10005
)

func vendorSlug(vendorID, deviceType uint32, name string) string {
	lower := strings.ToLower(name)
	if deviceType == physicalDeviceTypeCPU || strings.Contains(lower, "llvmpipe") {
		if strings.Contains(lower, "llvmpipe") {
			return "llvmpipe"
		}
		return "cpu"
	}
	switch vendorID {
	case vendorIDAMD:
		return "amd"
	case vendorIDNVIDIA:
		return "nvidia"
	case vendorIDIntel:
		return "intel"
	case vendorIDARM:
		return "arm"
	case vendorIDQualcomm:
		return "qualcomm"
	case vendorIDApple:
		return "apple"
	case vendorIDMesa:
		return "mesa"
	}
	switch {
	case strings.Contains(lower, "nvidia"):
		return "nvidia"
	case strings.Contains(lower, "amd") || strings.Contains(lower, "radeon") || strings.Contains(lower, "radv"):
		return "amd"
	case strings.Contains(lower, "intel"):
		return "intel"
	case strings.Contains(lower, "apple") || strings.Contains(lower, "moltenvk"):
		return "apple"
	}
	return "unknown"
}
