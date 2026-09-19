package vulkan

import (
	"fmt"
	"strings"
)

// DeviceType is VkPhysicalDeviceType. Values match the spec so a raw
// properties.deviceType casts. String is the CLI / driver-id token.
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

// ParseDeviceType maps a String() token such as dedicated or software.
func ParseDeviceType(name string) (DeviceType, error) {
	var zero DeviceType
	for _, v := range zero.Values() {
		if v.String() == name {
			return v, nil
		}
	}
	return DeviceTypeOther, fmt.Errorf("%w: %q", ErrDeviceType, name)
}

func (t DeviceType) String() string {
	switch t {
	case DeviceTypeIntegrated:
		return "integrated"
	case DeviceTypeDedicated:
		return "dedicated"
	case DeviceTypeVirtual:
		return "virtual"
	case DeviceTypeSoftware:
		return "software"
	default:
		return "other"
	}
}

// Values lists DeviceType members for [github.com/lewtec/lewkit/x/cmd.EnumArg].
func (DeviceType) Values() []DeviceType {
	return []DeviceType{DeviceTypeOther, DeviceTypeIntegrated, DeviceTypeDedicated, DeviceTypeVirtual, DeviceTypeSoftware}
}

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
