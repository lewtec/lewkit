package vulkan

import "strings"

// Kind is the device class. Values match VkPhysicalDeviceType so a raw
// properties.deviceType casts. String is the CLI / driver-id token.
type Kind uint32

const (
	KindOther      Kind = 0
	KindIntegrated Kind = 1
	KindDedicated  Kind = 2
	KindVirtual    Kind = 3
	KindSoftware   Kind = 4
)

func (k Kind) String() string {
	switch k {
	case KindIntegrated:
		return "integrated"
	case KindDedicated:
		return "dedicated"
	case KindVirtual:
		return "virtual"
	case KindSoftware:
		return "software"
	default:
		return "other"
	}
}

// Values lists Kind members for [github.com/lewtec/lewkit/x/cmd.EnumArg].
func (Kind) Values() []Kind {
	return []Kind{KindOther, KindIntegrated, KindDedicated, KindVirtual, KindSoftware}
}

// Weight is the default driver weight for this class.
func (k Kind) Weight() int {
	switch k {
	case KindDedicated:
		return 70
	case KindIntegrated:
		return 50
	case KindVirtual:
		return 20
	case KindSoftware:
		return 0
	default:
		return 40
	}
}

func kindFrom(deviceType uint32, name string) Kind {
	if strings.Contains(strings.ToLower(name), "llvmpipe") {
		return KindSoftware
	}
	k := Kind(deviceType)
	switch k {
	case KindIntegrated, KindDedicated, KindVirtual, KindSoftware:
		return k
	default:
		return KindOther
	}
}
