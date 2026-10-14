package vulkan

import "strings"

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum type.go

// DeviceType is VkPhysicalDeviceType. Values match the spec so a raw
// properties.deviceType casts. String is the CLI / driver-id token.
// A value outside the set is other.
//
//lewkit:enum invalid=other values=other,integrated,dedicated,virtual,software
type DeviceType uint32

// Uint32 is the VkPhysicalDeviceType value.
func (t DeviceType) Uint32() uint32 { return uint32(t) }

// Weight is the default driver weight for this device type.
func (t DeviceType) Weight() int {
	switch t {
	case DeviceTypeDedicated:
		return 70
	case DeviceTypeIntegrated:
		return 50
	case DeviceTypeVirtual:
		return 20
	case DeviceTypeSoftware:
		return 0
	default:
		return 40
	}
}

func deviceTypeFrom(deviceType uint32, name string) DeviceType {
	if strings.Contains(strings.ToLower(name), "llvmpipe") {
		return DeviceTypeSoftware
	}
	t := DeviceType(deviceType)
	switch t {
	case DeviceTypeIntegrated, DeviceTypeDedicated, DeviceTypeVirtual, DeviceTypeSoftware:
		return t
	default:
		return DeviceTypeOther
	}
}
