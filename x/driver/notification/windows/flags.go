package windows

const (
	niifInfo    = 0x1
	niifError   = 0x3
	niifNoSound = 0x10
)

func infoFlags(urgency string) uint32 {
	switch urgency {
	case "critical":
		return niifError
	case "low":
		return niifInfo | niifNoSound
	default:
		return niifInfo
	}
}
