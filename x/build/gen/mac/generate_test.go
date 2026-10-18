package mac

import (
	"encoding/json"
	"fmt"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		"Sources/AppMain.swift",
		"Assets.xcassets/Contents.json",
		"Assets.xcassets/AppIcon.appiconset/Contents.json",
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
	if !strings.Contains(s, "ENABLE_APP_SANDBOX: NO") {
		require.FailNowf(t, "", "sandbox not off:\n%s", s)
	}
	if !strings.Contains(s, "ASSETCATALOG_COMPILER_APPICON_NAME: AppIcon") {
		require.FailNowf(t, "", "app icon name missing:\n%s", s)
	}
	if !strings.Contains(s, "Assets.xcassets") {
		require.FailNowf(t, "", "asset catalog missing:\n%s", s)
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
	if !strings.Contains(ps, "<key>CFBundleIconName</key>") || !strings.Contains(ps, "<string>AppIcon</string>") {
		require.FailNowf(t, "", "plist icon name:\n%s", ps)
	}

	jsonb, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	require.NoError(t, err)
	if !strings.Contains(string(jsonb), `"package_id": "br.tec.lew.counter"`) {
		require.FailNowf(t, "", "json: %s", jsonb)
	}

	swift, err := os.ReadFile(filepath.Join(out, "Sources/AppMain.swift"))
	require.NoError(t, err)
	if !strings.Contains(string(swift), "eletrocromo-server") || !strings.Contains(string(swift), "execv") {
		require.FailNowf(t, "", "launcher missing:\n%s", swift)
	}
	if strings.Contains(string(swift), "ELETROCROMO_NO_UI") || strings.Contains(string(swift), "WKWebView") {
		require.FailNowf(t, "", "launcher still owns a window:\n%s", swift)
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
}

func TestApplyMacIcons(t *testing.T) {
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
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "in.png")
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.SetNRGBA(x, y, color.NRGBA{R: 20, G: 40, B: 60, A: 255})
		}
	}
	f, err := os.Create(src)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, img))
	require.NoError(t, f.Close())
	iconRoot := filepath.Join(srcDir, "icons")
	if _, err := icons.Generate(icons.Options{SourcePath: src, OutputDir: iconRoot, Force: true}); err != nil {
		require.NoError(t, err)
	}
	assets := filepath.Join(out, "Assets.xcassets")
	require.NoError(t, applyMacIcons(iconRoot, assets))
	raw, err := os.ReadFile(filepath.Join(assets, "AppIcon.appiconset", "Contents.json"))
	require.NoError(t, err)
	var catalog struct {
		Images []struct {
			Filename string `json:"filename"`
			Size     string `json:"size"`
			Scale    string `json:"scale"`
		} `json:"images"`
	}
	require.NoError(t, json.Unmarshal(raw, &catalog))
	if len(catalog.Images) != len(macIconSlots) {
		require.FailNowf(t, "", "slots %d catalog %d", len(macIconSlots), len(catalog.Images))
	}
	for _, slot := range catalog.Images {
		pngFile, err := os.Open(filepath.Join(assets, "AppIcon.appiconset", slot.Filename))
		require.NoError(t, err)
		cfg, err := png.DecodeConfig(pngFile)
		closeErr := pngFile.Close()
		require.NoError(t, err)
		if closeErr != nil {
			require.FailNow(t, fmt.Sprint(closeErr))
		}
		parts := strings.Split(slot.Size, "x")
		if len(parts) != 2 {
			require.FailNowf(t, "", "size %q", slot.Size)
		}
		points, err := strconv.Atoi(parts[0])
		require.NoError(t, err)
		scale, err := strconv.Atoi(strings.TrimSuffix(slot.Scale, "x"))
		require.NoError(t, err)
		if cfg.Width != points*scale || cfg.Height != points*scale {
			require.FailNowf(t, "", "%s: got %dx%d want %d", slot.Filename, cfg.Width, cfg.Height, points*scale)
		}
	}
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(t.Context(), Options{OutDir: t.TempDir(), Config: Config{PackageID: "Not an id"}})
	require.Error(t, err, "expected error")
}
