package apk

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

func TestCreate_PackageIDLayout(t *testing.T) {
	out := t.TempDir()
	err := Create(Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    "../../examples/counter",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	mustExist := []string{
		"eletrocromo.json",
		"settings.gradle.kts",
		"app/build.gradle.kts",
		"app/src/main/AndroidManifest.xml",
		"app/src/main/java/br/tec/lew/counter/MainActivity.kt",
		"app/src/main/java/br/tec/lew/counter/PageActivity.kt",
		"app/src/main/java/br/tec/lew/counter/Windows.kt",
		"app/src/main/java/br/tec/lew/counter/ServerService.kt",
		"app/src/main/res/xml/network_security_config.xml",
		"app/src/main/res/layout/activity_main.xml",
		"scripts/build-go.sh",
		"README.md",
	}
	for _, rel := range mustExist {
		p := filepath.Join(out, rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}

	gradle, err := os.ReadFile(filepath.Join(out, "app/build.gradle.kts"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(gradle)
	if !strings.Contains(s, `applicationId = "br.tec.lew.counter"`) {
		t.Fatalf("applicationId not baked in:\n%s", s)
	}
	if !strings.Contains(s, `namespace = "br.tec.lew.counter"`) {
		t.Fatalf("namespace not baked in:\n%s", s)
	}

	mainKt, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/MainActivity.kt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(mainKt), "package br.tec.lew.counter\n") {
		t.Fatalf("kotlin package mismatch:\n%s", mainKt[:80])
	}

	cfg, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), `"package_id": "br.tec.lew.counter"`) {
		t.Fatalf("json: %s", cfg)
	}
	if !strings.Contains(string(cfg), `"go_main": "../../examples/counter"`) {
		t.Fatalf("go_main: %s", cfg)
	}

	manifest, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(manifest), `android:scheme=`) {
		t.Fatalf("unexpected scheme filter:\n%s", manifest)
	}
	if !strings.Contains(string(manifest), `android.permission.POST_NOTIFICATIONS`) {
		t.Fatalf("missing notification permission:\n%s", manifest)
	}
	host, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Host.kt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(host), "fun notify(") {
		t.Fatal("host notify missing")
	}

	sh, err := os.ReadFile(filepath.Join(out, "scripts/build-go.sh"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(out, "scripts/build-go.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("build-go.sh not executable: %v", info.Mode())
	}
	if !strings.Contains(string(sh), "GOOS=android") {
		t.Fatalf("script missing GOOS=android")
	}
}

func TestCreate_CapabilitiesIntentFilters(t *testing.T) {
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
	manifest, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(manifest)
	if !strings.Contains(s, `android:scheme="myapp"`) {
		t.Fatalf("scheme:\n%s", s)
	}
	if !strings.Contains(s, `android:mimeType="text/markdown"`) {
		t.Fatalf("mime:\n%s", s)
	}
	if _, err := os.Stat(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/OpenDrop.kt")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/ShareOut.kt")); err != nil {
		t.Fatal(err)
	}
	man, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(man), "FileProvider") {
		t.Fatalf("FileProvider:\n%s", man)
	}
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(Options{
		OutDir: t.TempDir(),
		Config: Config{PackageID: "Not.Valid"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreate_RequiresForceWhenNonEmpty(t *testing.T) {
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Create(Options{
		OutDir: out,
		Config: Config{PackageID: "br.tec.lew.x"},
	})
	if err == nil {
		t.Fatal("expected non-empty error")
	}
	if err := Create(Options{
		OutDir: out,
		Force:  true,
		Config: Config{PackageID: "br.tec.lew.x", AppName: "X"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "keep")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("force should wipe old files, keep still there: %v", err)
	}
}

func TestCreate_DefaultAppNameFromID(t *testing.T) {
	out := t.TempDir()
	if err := Create(Options{
		OutDir: out,
		Config: Config{PackageID: "br.tec.lew.myapp"},
	}); err != nil {
		t.Fatal(err)
	}
	stringsXML, err := os.ReadFile(filepath.Join(out, "app/src/main/res/values/strings.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stringsXML), ">myapp</string>") {
		t.Fatalf("default name: %s", stringsXML)
	}
}
