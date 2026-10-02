package mac

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/build/gen/common"
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
	if !strings.Contains(string(ui), "SiblingWindow") {
		t.Fatalf("new window leaves the app:\n%s", ui)
	}
	if !strings.Contains(string(ui), "javaScriptCanOpenWindowsAutomatically") {
		t.Fatalf("scripted window.open stays blocked:\n%s", ui)
	}
	if !strings.Contains(string(ui), "isReleasedWhenClosed") {
		t.Fatalf("popup window is released on close:\n%s", ui)
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

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(Options{OutDir: t.TempDir(), Config: Config{PackageID: "Not an id"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
