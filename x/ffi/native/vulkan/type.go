package vulkan

import "strings"

// DeviceType is VkPhysicalDeviceType. Values match the spec so a raw
// properties.deviceType casts. String is the CLI / driver-id token.
// A value outside the set is other.
//
//lewkit:enum invalid=other
type DeviceType uint32

const (
	DeviceTypeOther      DeviceType = 0
	DeviceTypeIntegrated DeviceType = 1
	DeviceTypeDedicated  DeviceType = 2
	DeviceTypeVirtual    DeviceType = 3
	DeviceTypeSoftware   DeviceType = 4
)

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
