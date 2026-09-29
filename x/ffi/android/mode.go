package android

// UI mode night bits from android.content.res.Configuration.
const (
	uiModeNightMask = 0x30
	uiModeNightNo   = 0x10
	uiModeNightYes  = 0x20
)

// UiModeManager night-mode settings.
const (
	nightModeNo  int32 = 1
	nightModeYes int32 = 2
)

// NightFromUIMode reads the effective night bit.
// ok is false when the mask is neither yes nor no.
func NightFromUIMode(uiMode int32) (dark bool, ok bool) {
	switch uiMode & uiModeNightMask {
	case uiModeNightYes:
		return true, true
	case uiModeNightNo:
		return false, true
	default:
		return false, false
	}
}

// NightFromSetting maps UiModeManager.getNightMode.
// Auto and custom have no effective value here.
func NightFromSetting(mode int32) (dark bool, ok bool) {
	switch mode {
	case nightModeYes:
		return true, true
	case nightModeNo:
		return false, true
	default:
		return false, false
	}
}
