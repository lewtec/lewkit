package apk

// Resource ids for the hand-built APK. lewkit generate androiddex compiles
// classes.dex against these values. Package 0x7f. The low 16 bits are the
// entry index inside the type.
const (
	resIDSplash         uint32 = 0x7f010000
	resIDSplashDetail   uint32 = 0x7f010001
	resIDSplashLogo     uint32 = 0x7f010002
	resIDSplashProgress uint32 = 0x7f010003
	resIDSplashRetry    uint32 = 0x7f010004
	resIDSplashStatus   uint32 = 0x7f010005

	resIDLayoutMain uint32 = 0x7f020000

	resIDIcon uint32 = 0x7f030000

	resIDAppName          uint32 = 0x7f040000
	resIDLoading          uint32 = 0x7f040001
	resIDRetry            uint32 = 0x7f040002
	resIDServerFailed     uint32 = 0x7f040003
	resIDWebviewError     uint32 = 0x7f040004
	resIDWebviewHTTPError uint32 = 0x7f040005

	resIDTheme uint32 = 0x7f050000

	resIDFilePaths uint32 = 0x7f060000
	resIDNetSec    uint32 = 0x7f060001
)

// Framework ids from the API 35 public resource table.
const (
	attrTheme                     uint32 = 0x01010000
	attrLabel                     uint32 = 0x01010001
	attrIcon                      uint32 = 0x01010002
	attrName                      uint32 = 0x01010003
	attrDebuggable                uint32 = 0x0101000f
	attrExported                  uint32 = 0x01010010
	attrAuthorities               uint32 = 0x01010018
	attrGrantUriPermissions       uint32 = 0x0101001b
	attrLaunchMode                uint32 = 0x0101001d
	attrConfigChanges             uint32 = 0x0101001f
	attrResource                  uint32 = 0x01010025
	attrMimeType                  uint32 = 0x01010026
	attrScheme                    uint32 = 0x01010027
	attrColorBackground           uint32 = 0x01010031
	attrTextAppearance            uint32 = 0x01010034
	attrTextColorSecondary        uint32 = 0x01010038
	attrTextAppearanceMedium      uint32 = 0x01010041
	attrTextAppearanceSmall       uint32 = 0x01010042
	attrWindowNoTitle             uint32 = 0x01010056
	attrProgressBarStyle          uint32 = 0x01010077
	attrGravity                   uint32 = 0x010100af
	attrOrientation               uint32 = 0x010100c4
	attrID                        uint32 = 0x010100d0
	attrBackground                uint32 = 0x010100d4
	attrVisibility                uint32 = 0x010100dc
	attrLayoutWidth               uint32 = 0x010100f4
	attrLayoutHeight              uint32 = 0x010100f5
	attrLayoutMarginTop           uint32 = 0x010100f8
	attrTextColor                 uint32 = 0x01010098
	attrSrc                       uint32 = 0x01010119
	attrScaleType                 uint32 = 0x0101011d
	attrText                      uint32 = 0x0101014f
	attrLayoutBelow               uint32 = 0x01010185
	attrLayoutCenterInParent      uint32 = 0x0101018f
	attrLayoutCenterHorizontal    uint32 = 0x01010190
	attrContentDescription        uint32 = 0x01010273
	attrMinSdkVersion             uint32 = 0x0101020c
	attrVersionCode               uint32 = 0x0101021b
	attrVersionName               uint32 = 0x0101021c
	attrTargetSdkVersion          uint32 = 0x01010270
	attrAllowBackup               uint32 = 0x01010280
	attrWindowActionBar           uint32 = 0x010102cd
	attrPaddingStart              uint32 = 0x010103b3
	attrPaddingEnd                uint32 = 0x010103b4
	attrSupportsRtl               uint32 = 0x010103af
	attrTextColorTertiary         uint32 = 0x01010212
	attrExtractNativeLibs         uint32 = 0x010104ea
	attrUsesCleartextTraffic      uint32 = 0x010104ec
	attrNetworkSecurityConfig     uint32 = 0x01010527
	attrCompileSdkVersion         uint32 = 0x01010572
	attrCompileSdkVersionCodename uint32 = 0x01010573

	styleDeviceDefaultNoActionBar uint32 = 0x01030129
	styleDeviceDefaultDayNight    uint32 = 0x010302e3
)

const (
	typeIntDec     byte = 0x10
	typeIntHex     byte = 0x11
	typeIntBoolean byte = 0x12
	typeString     byte = 0x03
	typeReference  byte = 0x01
	typeAttribute  byte = 0x02
	typeDimension  byte = 0x05
)

const (
	cfgChangeOrientation = 0x00000080
	cfgChangeKeyboard    = 0x00000020
	cfgChangeScreenSize  = 0x00000400
	cfgChangeUIMode      = 0x00000200
	cfgChangeFontScale   = 0x40000000
	cfgChanges           = cfgChangeOrientation | cfgChangeKeyboard | cfgChangeScreenSize | cfgChangeUIMode | cfgChangeFontScale

	launchSingleTask = 2
	launchStandard   = 0

	dimenMatch = 0xffffffff
	dimenWrap  = 0xfffffffe
	boolTrue   = 0xffffffff
	boolFalse  = 0
)

func androidAttrs() map[string]uint32 {
	return map[string]uint32{
		"allowBackup":               attrAllowBackup,
		"authorities":               attrAuthorities,
		"background":                attrBackground,
		"compileSdkVersion":         attrCompileSdkVersion,
		"compileSdkVersionCodename": attrCompileSdkVersionCodename,
		"configChanges":             attrConfigChanges,
		"contentDescription":        attrContentDescription,
		"debuggable":                attrDebuggable,
		"exported":                  attrExported,
		"extractNativeLibs":         attrExtractNativeLibs,
		"grantUriPermissions":       attrGrantUriPermissions,
		"gravity":                   attrGravity,
		"icon":                      attrIcon,
		"id":                        attrID,
		"label":                     attrLabel,
		"launchMode":                attrLaunchMode,
		"layout_below":              attrLayoutBelow,
		"layout_centerHorizontal":   attrLayoutCenterHorizontal,
		"layout_centerInParent":     attrLayoutCenterInParent,
		"layout_height":             attrLayoutHeight,
		"layout_marginTop":          attrLayoutMarginTop,
		"layout_width":              attrLayoutWidth,
		"mimeType":                  attrMimeType,
		"minSdkVersion":             attrMinSdkVersion,
		"name":                      attrName,
		"networkSecurityConfig":     attrNetworkSecurityConfig,
		"orientation":               attrOrientation,
		"paddingEnd":                attrPaddingEnd,
		"paddingStart":              attrPaddingStart,
		"resource":                  attrResource,
		"scaleType":                 attrScaleType,
		"scheme":                    attrScheme,
		"src":                       attrSrc,
		"supportsRtl":               attrSupportsRtl,
		"targetSdkVersion":          attrTargetSdkVersion,
		"text":                      attrText,
		"textAppearance":            attrTextAppearance,
		"textColor":                 attrTextColor,
		"theme":                     attrTheme,
		"usesCleartextTraffic":      attrUsesCleartextTraffic,
		"versionCode":               attrVersionCode,
		"versionName":               attrVersionName,
		"visibility":                attrVisibility,
	}
}
