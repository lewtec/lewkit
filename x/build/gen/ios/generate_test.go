package ios

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

func TestExcludedArch(t *testing.T) {
	t.Parallel()
	if got := excludedArch("arm64"); got != "x86_64" {
		t.Fatalf("arm64: got %q", got)
	}
	if got := excludedArch("x86_64"); got != "arm64" {
		t.Fatalf("x86_64: got %q", got)
	}
}

func TestNormalizeSDK(t *testing.T) {
	t.Parallel()
	got, err := normalizeSDK("")
	if err != nil || got != SDKSimulator {
		t.Fatalf("empty: got %q %v", got, err)
	}
	got, err = normalizeSDK("simulator")
	if err != nil || got != SDKSimulator {
		t.Fatalf("simulator: got %q %v", got, err)
	}
	got, err = normalizeSDK("device")
	if err != nil || got != SDKDevice {
		t.Fatalf("device: got %q %v", got, err)
	}
	if _, err := normalizeSDK("watchos"); err == nil {
		t.Fatal("expected error for watchos")
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
	if err != nil {
		t.Fatal(err)
	}

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
			t.Errorf("missing %s: %v", rel, err)
		}
	}

	yml, err := os.ReadFile(filepath.Join(out, "project.yml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(yml)
	if !strings.Contains(s, "PRODUCT_BUNDLE_IDENTIFIER: br.tec.lew.counter") {
		t.Fatalf("bundle id missing:\n%s", s)
	}
	if !strings.Contains(s, "platform: iOS") {
		t.Fatalf("platform missing:\n%s", s)
	}
	if !strings.Contains(s, "TARGETED_DEVICE_FAMILY: \"1,2\"") {
		t.Fatalf("device family missing:\n%s", s)
	}
	if !strings.Contains(s, "SWIFT_OBJC_BRIDGING_HEADER") {
		t.Fatalf("bridging header missing:\n%s", s)
	}
	if !strings.Contains(s, "ENABLE_DEBUG_DYLIB: NO") {
		t.Fatalf("debug dylib not off:\n%s", s)
	}
	if !strings.Contains(s, "LaunchScreen.storyboard") {
		t.Fatalf("launch storyboard missing from project:\n%s", s)
	}
	if strings.Contains(s, "type: app-extension") {
		t.Fatalf("share extension should be off without files:\n%s", s)
	}
	if !strings.Contains(s, "CODE_SIGNING_ALLOWED: YES") {
		t.Fatalf("ad-hoc signing required for app groups:\n%s", s)
	}
	if !strings.Contains(s, "-framework Metal") || !strings.Contains(s, "-framework QuartzCore") {
		t.Fatalf("metal frameworks missing:\n%s", s)
	}

	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	ps := string(plist)
	if !strings.Contains(ps, "br.tec.lew.counter") {
		t.Fatalf("plist id:\n%s", ps)
	}
	if !strings.Contains(ps, "NSAllowsLocalNetworking") {
		t.Fatalf("plist ATS:\n%s", ps)
	}
	if !strings.Contains(ps, "LSRequiresIPhoneOS") {
		t.Fatalf("plist iPhoneOS:\n%s", ps)
	}
	if !strings.Contains(ps, "UILaunchStoryboardName") || !strings.Contains(ps, "LaunchScreen") {
		t.Fatalf("plist launch storyboard:\n%s", ps)
	}
	if strings.Contains(ps, "UILaunchScreen") {
		t.Fatalf("plist still uses UILaunchScreen (zooms 1x 1024px):\n%s", ps)
	}

	story, err := os.ReadFile(filepath.Join(out, "LaunchScreen.storyboard"))
	if err != nil {
		t.Fatal(err)
	}
	ssb := string(story)
	if !strings.Contains(ssb, `constant="120"`) || !strings.Contains(ssb, "SplashLogo") {
		t.Fatalf("launch storyboard logo size:\n%s", ssb)
	}

	jsonb, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonb), `"package_id": "br.tec.lew.counter"`) {
		t.Fatalf("json: %s", jsonb)
	}
	if !strings.Contains(string(jsonb), `"generator": "eletrocromo-ios"`) {
		t.Fatalf("generator: %s", jsonb)
	}

	swift, err := os.ReadFile(filepath.Join(out, "Sources/ServerProcess.swift"))
	if err != nil {
		t.Fatal(err)
	}
	ss := string(swift)
	if !strings.Contains(ss, "ELETROCROMO_READY") {
		t.Fatalf("ready prefix missing:\n%s", ss)
	}
	if !strings.Contains(ss, "EletrocromoStart") {
		t.Fatalf("c-archive entry missing:\n%s", ss)
	}
	if !strings.Contains(ss, "hostsNativeSurface") {
		t.Fatalf("native surface must skip the ready timeout:\n%s", ss)
	}
	if !strings.Contains(ss, "dirs.cache.path") {
		t.Fatalf("start must pass cache dir into Go:\n%s", ss)
	}
	if !strings.Contains(ss, "AskWatch.start") {
		t.Fatal("ios host does not watch the ask directory")
	}
	if !strings.Contains(ss, "HostWatch.start") {
		t.Fatal("ios host does not watch the ios directory")
	}
	ask, err := os.ReadFile(filepath.Join(out, "Sources/AskWatch.swift"))
	if err != nil {
		t.Fatal(err)
	}
	sheet := string(ask)
	if strings.Contains(sheet, "DispatchQueue.main.sync") || strings.Contains(sheet, "RunLoop.current.run") {
		t.Fatal("ios ask blocks the main run loop")
	}
	if !strings.Contains(sheet, "removeItem") || !strings.Contains(sheet, ".actionSheet") {
		t.Fatal("ios chooser still replays a killed request as an alert")
	}
	host, err := os.ReadFile(filepath.Join(out, "Sources/HostWatch.swift"))
	if err != nil {
		t.Fatal(err)
	}
	hs := string(host)
	if strings.Contains(hs, "DispatchQueue.main.sync") || strings.Contains(hs, "RunLoop.current.run") {
		t.Fatal("ios host blocks the main run loop")
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
			t.Errorf("HostWatch missing %s", needle)
		}
	}

	ui, err := os.ReadFile(filepath.Join(out, "Sources/RootViewController.swift"))
	if err != nil {
		t.Fatal(err)
	}
	us := string(ui)
	if !strings.Contains(us, "WKWebView") {
		t.Fatalf("webview missing:\n%s", us)
	}
	if !strings.Contains(us, "UIRefreshControl") {
		t.Fatalf("pull-to-refresh missing:\n%s", us)
	}
	if !strings.Contains(us, "SplashLogo") {
		t.Fatalf("splash logo missing:\n%s", us)
	}
	if !strings.Contains(us, "Try again") {
		t.Fatalf("android retry copy missing:\n%s", us)
	}
	if !strings.Contains(us, "openExternal") {
		t.Fatalf("custom scheme open missing:\n%s", us)
	}
	if !strings.Contains(us, "func openSurface()") {
		t.Fatalf("native surface missing:\n%s", us)
	}
	if !strings.Contains(us, "EletrocromoPointer") || !strings.Contains(us, "EletrocromoResize") {
		t.Fatalf("surface events missing:\n%s", us)
	}
	if !strings.Contains(us, "revealIfStuck") {
		t.Fatalf("stuck reveal missing:\n%s", us)
	}
	if !strings.Contains(us, "publishDaynight") {
		t.Fatalf("appearance publish missing:\n%s", us)
	}
	if strings.Contains(us, "UIBarButtonItem") || strings.Contains(us, "arrow.clockwise") {
		t.Fatalf("navbar reload still present:\n%s", us)
	}
	if !strings.Contains(us, "UIApplication.shared.open") {
		t.Fatalf("off-loopback open missing:\n%s", us)
	}

	delegate, err := os.ReadFile(filepath.Join(out, "Sources/AppDelegate.swift"))
	if err != nil {
		t.Fatal(err)
	}
	ds := string(delegate)
	if strings.Contains(ds, "UINavigationController") {
		t.Fatalf("nav controller still wrapping root:\n%s", ds)
	}
	if !strings.Contains(ds, "quietSplash") {
		t.Fatalf("quiet splash missing:\n%s", ds)
	}
	if !strings.Contains(ds, "applicationDidBecomeActive") {
		t.Fatalf("become-active drain missing:\n%s", ds)
	}
	if !strings.Contains(ds, "HostWatch.prepare") {
		t.Fatalf("ios host state is not published at launch:\n%s", ds)
	}

	hdr, err := os.ReadFile(filepath.Join(out, "Sources/eletrocromo-Bridging-Header.h"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(hdr), "libeletrocromo.h") {
		t.Fatalf("header: %s", hdr)
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
	if err != nil {
		t.Fatal(err)
	}
	plist, err := os.ReadFile(filepath.Join(out, "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	ps := string(plist)
	if !strings.Contains(ps, "CFBundleURLTypes") || !strings.Contains(ps, "myapp") {
		t.Fatalf("url types:\n%s", ps)
	}
	if !strings.Contains(ps, "CFBundleDocumentTypes") {
		t.Fatalf("docs:\n%s", ps)
	}
	if !strings.Contains(ps, "UTImportedTypeDeclarations") || !strings.Contains(ps, "LSHandlerRank") {
		t.Fatalf("uti:\n%s", ps)
	}
	yml, err := os.ReadFile(filepath.Join(out, "project.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(yml), "type: app-extension") {
		t.Fatalf("share extension missing:\n%s", yml)
	}
	if _, err := os.Stat(filepath.Join(out, "ShareExtension/ShareViewController.swift")); err != nil {
		t.Fatal(err)
	}
	extPlist, err := os.ReadFile(filepath.Join(out, "ShareExtension/Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(extPlist), "NSExtensionActivationSupportsImageWithMaxCount") {
		t.Fatalf("image activation:\n%s", extPlist)
	}
	if _, err := os.Stat(filepath.Join(out, "Sources/OpenDrop.swift")); err != nil {
		t.Fatal(err)
	}
	openDrop, err := os.ReadFile(filepath.Join(out, "Sources/OpenDrop.swift"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(openDrop), "adoptGroupLine") {
		t.Fatalf("drain must copy group files into Cache/inbox:\n%s", openDrop)
	}
	shareSrc, err := os.ReadFile(filepath.Join(out, "ShareExtension/ShareViewController.swift"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(shareSrc), "loadFileRepresentation") {
		t.Fatalf("photos share needs loadFileRepresentation:\n%s", shareSrc)
	}
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(t.Context(), Options{OutDir: t.TempDir(), Config: Config{PackageID: "Not an id"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBridgeSource_ExportsStart(t *testing.T) {
	t.Parallel()
	if !strings.Contains(iosBridgeSource, "//export EletrocromoStart") {
		t.Fatal("missing export")
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_NO_UI") {
		t.Fatal("missing NO_UI")
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_READY_FILE") {
		t.Fatal("missing READY_FILE")
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_CACHE_DIR") {
		t.Fatal("missing CACHE_DIR")
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_ASK_DIR") {
		t.Fatal("missing ASK_DIR")
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_DATA_DIR") {
		t.Fatal("missing DATA_DIR")
	}
	if !strings.Contains(iosBridgeSource, "ELETROCROMO_CONFIG_DIR") {
		t.Fatal("missing CONFIG_DIR")
	}
	if !strings.Contains(iosBridgeSource, "main()") {
		t.Fatal("missing main() call")
	}
	if !strings.Contains(iosBridgeSource, "//export EletrocromoPointer") {
		t.Fatal("missing pointer export")
	}
	if !strings.Contains(iosBridgeSource, "//export EletrocromoResize") {
		t.Fatal("missing resize export")
	}
	if !strings.Contains(iosBridgeSource, "//export EletrocromoSurfaceLost") {
		t.Fatal("missing surface-lost export")
	}
	if !strings.Contains(iosBridgeSource, "entry.DeliverPointer") {
		t.Fatal("pointer export must deliver")
	}
}
