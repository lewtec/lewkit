package mac

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/icons"
)

func TestCreate_WritesHost(t *testing.T) {
	out := t.TempDir()
	err := Create(Options{
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
		"Sources/ServerProcess.swift",
		"Sources/MainWindow.swift",
		"Assets.xcassets/Contents.json",
		"Assets.xcassets/AppIcon.appiconset/Contents.json",
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
	if !strings.Contains(s, "ENABLE_APP_SANDBOX: NO") {
		t.Fatalf("sandbox not off:\n%s", s)
	}
	if !strings.Contains(s, "ASSETCATALOG_COMPILER_APPICON_NAME: AppIcon") {
		t.Fatalf("app icon name missing:\n%s", s)
	}
	if !strings.Contains(s, "Assets.xcassets") {
		t.Fatalf("asset catalog missing:\n%s", s)
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
	if !strings.Contains(ps, "<key>CFBundleIconName</key>") || !strings.Contains(ps, "<string>AppIcon</string>") {
		t.Fatalf("plist icon name:\n%s", ps)
	}

	jsonb, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jsonb), `"package_id": "br.tec.lew.counter"`) {
		t.Fatalf("json: %s", jsonb)
	}

	swift, err := os.ReadFile(filepath.Join(out, "Sources/ServerProcess.swift"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(swift), "ELETROCROMO_NO_UI") {
		t.Fatalf("helper env missing:\n%s", swift)
	}
	if !strings.Contains(string(swift), "ELETROCROMO_ASK_DIR") || !strings.Contains(string(swift), "AskWatch.start") {
		t.Fatal("mac host does not watch the ask directory")
	}

	ui, err := os.ReadFile(filepath.Join(out, "Sources/MainWindow.swift"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ui), "NSTitlebarAccessoryViewController") {
		t.Fatalf("titlebar reload missing:\n%s", ui)
	}
	if !strings.Contains(string(ui), "arrow.clockwise") {
		t.Fatalf("reload symbol missing:\n%s", ui)
	}
	if !strings.Contains(string(ui), "openExternal") {
		t.Fatalf("custom scheme open missing:\n%s", ui)
	}
}

func TestCreate_CapabilitiesPlist(t *testing.T) {
	out := t.TempDir()
	err := Create(Options{
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
	if _, err := os.Stat(filepath.Join(out, "Sources/OpenDrop.swift")); err != nil {
		t.Fatal(err)
	}
}

func TestApplyMacIcons(t *testing.T) {
	out := t.TempDir()
	err := Create(Options{
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
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "in.png")
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.SetNRGBA(x, y, color.NRGBA{R: 20, G: 40, B: 60, A: 255})
		}
	}
	f, err := os.Create(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	iconRoot := filepath.Join(srcDir, "icons")
	if _, err := icons.Generate(icons.Options{SourcePath: src, OutputDir: iconRoot, Force: true}); err != nil {
		t.Fatal(err)
	}
	assets := filepath.Join(out, "Assets.xcassets")
	if err := applyMacIcons(iconRoot, assets); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(assets, "AppIcon.appiconset", "Contents.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Images []struct {
			Filename string `json:"filename"`
			Size     string `json:"size"`
			Scale    string `json:"scale"`
		} `json:"images"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	if len(catalog.Images) != len(macIconSlots) {
		t.Fatalf("slots %d catalog %d", len(macIconSlots), len(catalog.Images))
	}
	for _, slot := range catalog.Images {
		pngFile, err := os.Open(filepath.Join(assets, "AppIcon.appiconset", slot.Filename))
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(pngFile)
		closeErr := pngFile.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		parts := strings.Split(slot.Size, "x")
		if len(parts) != 2 {
			t.Fatalf("size %q", slot.Size)
		}
		points, err := strconv.Atoi(parts[0])
		if err != nil {
			t.Fatal(err)
		}
		scale, err := strconv.Atoi(strings.TrimSuffix(slot.Scale, "x"))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Width != points*scale || cfg.Height != points*scale {
			t.Fatalf("%s: got %dx%d want %d", slot.Filename, cfg.Width, cfg.Height, points*scale)
		}
	}
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(Options{OutDir: t.TempDir(), Config: Config{PackageID: "Not an id"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
