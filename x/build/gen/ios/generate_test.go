package ios

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIOSSimulator(t *testing.T) {
	t.Parallel()
	device, err := iosSimulator("", "arm64")
	require.NoError(t, err)
	require.False(t, device)
	sim, err := iosSimulator("", "amd64")
	require.NoError(t, err)
	require.True(t, sim)
	sim, err = iosSimulator("iphonesimulator", "arm64")
	require.NoError(t, err)
	require.True(t, sim)
	device, err = iosSimulator("iphoneos", "arm64")
	require.NoError(t, err)
	require.False(t, device)
	_, err = iosSimulator("watchos", "arm64")
	require.Error(t, err)
}

func TestNormalizeSDK(t *testing.T) {
	t.Parallel()
	got, err := normalizeSDK("")
	if err != nil || got != string(SDKSimulator) {
		require.FailNowf(t, "", "empty: got %q %v", got, err)
	}
	got, err = normalizeSDK("simulator")
	if err != nil || got != string(SDKSimulator) {
		require.FailNowf(t, "", "simulator: got %q %v", got, err)
	}
	got, err = normalizeSDK("device")
	if err != nil || got != string(SDKDevice) {
		require.FailNowf(t, "", "device: got %q %v", got, err)
	}
	if _, err := normalizeSDK("watchos"); err == nil {
		require.FailNow(t, fmt.Sprint("expected error for watchos"))
	}
}

func TestCreate_WritesHost(t *testing.T) {
	out := t.TempDir()
	err := Create(t.Context(), Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    ".",
		},
	})
	require.NoError(t, err)

	mustExist := []string{
		"eletrocromo.json",
		"project.yml",
		"Info.plist",
		"README.md",
		"Sources/AppDelegate.swift",
		"Sources/AskWatch.swift",
		"Sources/HostWatch.swift",
		"Sources/ServerProcess.swift",
		"Sources/RootViewController.swift",
		"Sources/eletrocromo-Bridging-Header.h",
		"Assets.xcassets/AppIcon.appiconset/Contents.json",
		"Assets.xcassets/SplashLogo.imageset/Contents.json",
		"LaunchScreen.storyboard",
	}
	for _, rel := range mustExist {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			assert.Failf(t, "", "missing %s: %v", rel, err)
		}
	}

	yml, err := os.ReadFile(filepath.Join(out, "project.yml"))
	require.NoError(t, err)
	s := string(yml)
	if !strings.Contains(s, "PRODUCT_BUNDLE_IDENTIFIER: br.tec.lew.counter") {
		require.FailNowf(t, "", "bundle id missing:\n%s", s)
	}
	if !strings.Contains(s, "platform: iOS") {
		require.FailNowf(t, "", "platform missing:\n%s", s)
	}
	if !strings.Contains(s, "TARGETED_DEVICE_FAMILY: \"1,2\"") {
		require.FailNowf(t, "", "device family missing:\n%s", s)
	}
	if !strings.Contains(s, "SWIFT_OBJC_BRIDGING_HEADER") {
		require.FailNowf(t, "", "bridging header missing:\n%s", s)
	}
	if !strings.Contains(s, "ENABLE_DEBUG_DYLIB: NO") {
		require.FailNowf(t, "", "debug dylib not off:\n%s", s)
	}
	if !strings.Contains(s, "LaunchScreen.storyboard") {
		require.FailNowf(t, "", "launch storyboard missing from project:\n%s", s)
	}
	if strings.Contains(s, "type: app-extension") {
		require.FailNowf(t, "", "share extension should be off without files:\n%s", s)
	}
	if !strings.Contains(s, "CODE_SIGNING_ALLOWED: YES") {
		require.FailNowf(t, "", "ad-hoc signing required for app groups:\n%s", s)
	}
	if !strings.Contains(s, "-framework Metal") || !strings.Contains(s, "-framework QuartzCore") {
		require.FailNowf(t, "", "metal frameworks missing:\n%s", s)
	}

	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	require.NoError(t, err)
	ps := string(plist)
	if !strings.Contains(ps, "br.tec.lew.counter") {
		require.FailNowf(t, "", "plist id:\n%s", ps)
	}
	if !strings.Contains(ps, "NSAllowsLocalNetworking") {
		require.FailNowf(t, "", "plist ATS:\n%s", ps)
	}
	if !strings.Contains(ps, "LSRequiresIPhoneOS") {
		require.FailNowf(t, "", "plist iPhoneOS:\n%s", ps)
	}
	if !strings.Contains(ps, "UILaunchStoryboardName") || !strings.Contains(ps, "LaunchScreen") {
		require.FailNowf(t, "", "plist launch storyboard:\n%s", ps)
	}
	if strings.Contains(ps, "UILaunchScreen") {
		require.FailNowf(t, "", "plist still uses UILaunchScreen (zooms 1x 1024px):\n%s", ps)
	}

	story, err := os.ReadFile(filepath.Join(out, "LaunchScreen.storyboard"))
	require.NoError(t, err)
	ssb := string(story)
	if !strings.Contains(ssb, `constant="120"`) || !strings.Contains(ssb, "SplashLogo") {
		require.FailNowf(t, "", "launch storyboard logo size:\n%s", ssb)
	}

	jsonb, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	require.NoError(t, err)
	if !strings.Contains(string(jsonb), `"package_id": "br.tec.lew.counter"`) {
		require.FailNowf(t, "", "json: %s", jsonb)
	}
	if !strings.Contains(string(jsonb), `"generator": "eletrocromo-ios"`) {
		require.FailNowf(t, "", "generator: %s", jsonb)
	}

	swift, err := os.ReadFile(filepath.Join(out, "Sources/ServerProcess.swift"))
	require.NoError(t, err)
	ss := string(swift)
	if !strings.Contains(ss, "ELETROCROMO_READY") {
		require.FailNowf(t, "", "ready prefix missing:\n%s", ss)
	}
	if !strings.Contains(ss, "EletrocromoStart") {
		require.FailNowf(t, "", "c-archive entry missing:\n%s", ss)
	}
	if !strings.Contains(ss, "hostsNativeSurface") {
		require.FailNowf(t, "", "native surface must skip the ready timeout:\n%s", ss)
	}
	if !strings.Contains(ss, "dirs.cache.path") {
		require.FailNowf(t, "", "start must pass cache dir into Go:\n%s", ss)
	}
	if !strings.Contains(ss, "AskWatch.start") {
		require.FailNow(t, fmt.Sprint("ios host does not watch the ask directory"))
	}
	if !strings.Contains(ss, "HostWatch.start") {
		require.FailNow(t, fmt.Sprint("ios host does not watch the ios directory"))
	}
	ask, err := os.ReadFile(filepath.Join(out, "Sources/AskWatch.swift"))
	require.NoError(t, err)
	sheet := string(ask)
	if strings.Contains(sheet, "DispatchQueue.main.sync") || strings.Contains(sheet, "RunLoop.current.run") {
		require.FailNow(t, fmt.Sprint("ios ask blocks the main run loop"))
	}
	if !strings.Contains(sheet, "removeItem") || !strings.Contains(sheet, ".actionSheet") {
		require.FailNow(t, fmt.Sprint("ios chooser still replays a killed request as an alert"))
	}
	host, err := os.ReadFile(filepath.Join(out, "Sources/HostWatch.swift"))
	require.NoError(t, err)
	hs := string(host)
	if strings.Contains(hs, "DispatchQueue.main.sync") || strings.Contains(hs, "RunLoop.current.run") {
		require.FailNow(t, fmt.Sprint("ios host blocks the main run loop"))
	}
	for _, needle := range []string{
		"UIDocumentPickerViewController",
		"UIPasteboard",
		"UNUserNotificationCenter",
		"isBatteryMonitoringEnabled",
		"watchBattery",
		"probeBattery",
		"UIApplication.shared.open",
		"battery.json",
		"daynight.txt",
		"removeItem",
	} {
		if !strings.Contains(hs, needle) {
			assert.Failf(t, "", "HostWatch missing %s", needle)
		}
	}

	ui, err := os.ReadFile(filepath.Join(out, "Sources/RootViewController.swift"))
	require.NoError(t, err)
	us := string(ui)
	if !strings.Contains(us, "WKWebView") {
		require.FailNowf(t, "", "webview missing:\n%s", us)
	}
	if !strings.Contains(us, "UIRefreshControl") {
		require.FailNowf(t, "", "pull-to-refresh missing:\n%s", us)
	}
	if !strings.Contains(us, "SplashLogo") {
		require.FailNowf(t, "", "splash logo missing:\n%s", us)
	}
	if !strings.Contains(us, "Try again") {
		require.FailNowf(t, "", "android retry copy missing:\n%s", us)
	}
	if !strings.Contains(us, "openExternal") {
		require.FailNowf(t, "", "custom scheme open missing:\n%s", us)
	}
	if !strings.Contains(us, "func openSurface()") {
		require.FailNowf(t, "", "native surface missing:\n%s", us)
	}
	if !strings.Contains(us, "EletrocromoPointer") || !strings.Contains(us, "EletrocromoResize") || !strings.Contains(us, "EletrocromoDead") {
		require.FailNowf(t, "", "surface events missing:\n%s", us)
	}
	if !strings.Contains(us, "revealIfStuck") {
		require.FailNowf(t, "", "stuck reveal missing:\n%s", us)
	}
	if !strings.Contains(us, "publishDaynight") {
		require.FailNowf(t, "", "appearance publish missing:\n%s", us)
	}
	if strings.Contains(us, "UIBarButtonItem") || strings.Contains(us, "arrow.clockwise") {
		require.FailNowf(t, "", "navbar reload still present:\n%s", us)
	}
	if !strings.Contains(us, "UIApplication.shared.open") {
		require.FailNowf(t, "", "off-loopback open missing:\n%s", us)
	}

	delegate, err := os.ReadFile(filepath.Join(out, "Sources/AppDelegate.swift"))
	require.NoError(t, err)
	ds := string(delegate)
	if strings.Contains(ds, "UINavigationController") {
		require.FailNowf(t, "", "nav controller still wrapping root:\n%s", ds)
	}
	if !strings.Contains(ds, "quietSplash") {
		require.FailNowf(t, "", "quiet splash missing:\n%s", ds)
	}
	if !strings.Contains(ds, "applicationDidBecomeActive") {
		require.FailNowf(t, "", "become-active drain missing:\n%s", ds)
	}
	if !strings.Contains(ds, "HostWatch.prepare") {
		require.FailNowf(t, "", "ios host state is not published at launch:\n%s", ds)
	}

	hdr, err := os.ReadFile(filepath.Join(out, "Sources/eletrocromo-Bridging-Header.h"))
	require.NoError(t, err)
	if !strings.Contains(string(hdr), "libeletrocromo.h") {
		require.FailNowf(t, "", "header: %s", hdr)
	}
}

func TestCreate_CapabilitiesPlist(t *testing.T) {
	out := t.TempDir()
	err := Create(t.Context(), Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    ".",
			Capabilities: common.Capabilities{
				URL:   &common.URLCap{Schemes: []string{"myapp"}},
				Files: &common.FilesCap{Types: []common.FileType{{Ext: ".md", MIME: "text/markdown"}}},
			},
		},
	})
	require.NoError(t, err)
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	require.NoError(t, err)
	ps := string(plist)
	if !strings.Contains(ps, "CFBundleURLTypes") || !strings.Contains(ps, "myapp") {
		require.FailNowf(t, "", "url types:\n%s", ps)
	}
	if !strings.Contains(ps, "CFBundleDocumentTypes") {
		require.FailNowf(t, "", "docs:\n%s", ps)
	}
	if !strings.Contains(ps, "UTImportedTypeDeclarations") || !strings.Contains(ps, "LSHandlerRank") {
		require.FailNowf(t, "", "uti:\n%s", ps)
	}
	yml, err := os.ReadFile(filepath.Join(out, "project.yml"))
	require.NoError(t, err)
	if !strings.Contains(string(yml), "type: app-extension") {
		require.FailNowf(t, "", "share extension missing:\n%s", yml)
	}
	if _, err := os.Stat(filepath.Join(out, "ShareExtension/ShareViewController.swift")); err != nil {
		require.NoError(t, err)
	}
	extPlist, err := os.ReadFile(filepath.Join(out, "ShareExtension/Info.plist"))
	require.NoError(t, err)
	if !strings.Contains(string(extPlist), "NSExtensionActivationSupportsImageWithMaxCount") {
		require.FailNowf(t, "", "image activation:\n%s", extPlist)
	}
	if _, err := os.Stat(filepath.Join(out, "Sources/OpenDrop.swift")); err != nil {
		require.NoError(t, err)
	}
	openDrop, err := os.ReadFile(filepath.Join(out, "Sources/OpenDrop.swift"))
	require.NoError(t, err)
	if !strings.Contains(string(openDrop), "adoptGroupLine") {
		require.FailNowf(t, "", "drain must copy group files into Cache/inbox:\n%s", openDrop)
	}
	shareSrc, err := os.ReadFile(filepath.Join(out, "ShareExtension/ShareViewController.swift"))
	require.NoError(t, err)
	if !strings.Contains(string(shareSrc), "loadFileRepresentation") {
		require.FailNowf(t, "", "photos share needs loadFileRepresentation:\n%s", shareSrc)
	}
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(t.Context(), Options{OutDir: t.TempDir(), Config: Config{PackageID: "Not an id"}})
	require.Error(t, err, "expected error")
}

func TestBridgeSource_ExportsStart(t *testing.T) {
	t.Parallel()
	if !strings.Contains(iosBridgeSource, "//export EletrocromoStart") {
		require.FailNow(t, fmt.Sprint("missing export"))
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_NO_UI") {
		require.FailNow(t, fmt.Sprint("missing NO_UI"))
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_READY_FILE") {
		require.FailNow(t, fmt.Sprint("missing READY_FILE"))
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_CACHE_DIR") {
		require.FailNow(t, fmt.Sprint("missing CACHE_DIR"))
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_ASK_DIR") {
		require.FailNow(t, fmt.Sprint("missing ASK_DIR"))
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_DATA_DIR") {
		require.FailNow(t, fmt.Sprint("missing DATA_DIR"))
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_CONFIG_DIR") {
		require.FailNow(t, fmt.Sprint("missing CONFIG_DIR"))
	}
	if !strings.Contains(iosBridgeSource, "main()") {
		require.FailNow(t, fmt.Sprint("missing main() call"))
	}
	if !strings.Contains(iosBridgeSource, "//export EletrocromoPointer") {
		require.FailNow(t, fmt.Sprint("missing pointer export"))
	}
	if !strings.Contains(iosBridgeSource, "//export EletrocromoResize") || !strings.Contains(iosBridgeSource, "//export EletrocromoDead") {
		require.FailNow(t, fmt.Sprint("missing resize export"))
	}
	if !strings.Contains(iosBridgeSource, "//export EletrocromoSurfaceLost") {
		require.FailNow(t, fmt.Sprint("missing surface-lost export"))
	}
	if !strings.Contains(iosBridgeSource, "entry.DeliverPointer") {
		require.FailNow(t, fmt.Sprint("pointer export must deliver"))
	}
}
