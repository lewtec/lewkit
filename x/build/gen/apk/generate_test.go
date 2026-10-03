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
		"settings.gradle",
		"app/build.gradle",
		"app/src/main/AndroidManifest.xml",
		"app/src/main/java/br/tec/lew/counter/MainActivity.java",
		"app/src/main/java/lewkit/FileChooser.java",
		"app/src/main/java/lewkit/Documents.java",
		"app/src/main/java/br/tec/lew/counter/PageActivity.java",
		"app/src/main/java/br/tec/lew/counter/Windows.java",
		"app/src/main/java/br/tec/lew/counter/ServerService.java",
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

	gradle, err := os.ReadFile(filepath.Join(out, "app/build.gradle"))
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
	if !strings.Contains(s, "swiperefreshlayout:swiperefreshlayout:1.1.0") {
		t.Fatal("pull-to-refresh dependency missing")
	}
	if strings.Contains(s, "kotlin-stdlib") || strings.Contains(s, "appcompat") || strings.Contains(s, "androidx.core") || strings.Contains(s, "androidx.webkit") {
		t.Fatalf("kotlin or appcompat still on the classpath:\n%s", s)
	}

	mainJava, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/MainActivity.java"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(mainJava), "package br.tec.lew.counter;\n") {
		t.Fatalf("java package mismatch:\n%s", mainJava[:80])
	}
	if !strings.Contains(string(mainJava), "Host.noteForeground(this, true)") {
		t.Fatal("splash activity does not record the foreground window")
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
	if !strings.Contains(string(manifest), `android:configChanges="orientation|screenSize|keyboardHidden|uiMode"`) {
		t.Fatalf("manifest uiMode:\n%s", manifest)
	}
	if !strings.Contains(string(manifest), `android:name="lewkit.FileChooser"`) {
		t.Fatalf("file chooser activity:\n%s", manifest)
	}
	docs, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Documents.java"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(docs), "int readFd") {
		t.Fatal("document reader")
	}

	hostJava, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Host.java"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(hostJava), "dataDir") || strings.Contains(string(hostJava), "cacheDir") || strings.Contains(string(hostJava), "configDir") {
		t.Fatalf("host still owns directory methods:\n%s", hostJava)
	}
	if err := filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".kt") || strings.HasSuffix(path, ".kts") {
			t.Errorf("kotlin source %s", path)
		}
		if d.IsDir() || !(strings.HasSuffix(path, ".java") || strings.HasSuffix(path, ".xml") || strings.HasSuffix(path, ".gradle")) {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(body)
		for _, banned := range []string{"androidx.appcompat", "androidx.core", "androidx.activity", "androidx.webkit", "kotlin-stdlib"} {
			if strings.Contains(text, banned) {
				t.Errorf("%s still mentions %s", path, banned)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	proxy, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/GoProxy.java"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(proxy), "nativeInvoke") {
		t.Fatalf("GoProxy:\n%s", proxy)
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
	if _, err := os.Stat(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/OpenDrop.java")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/ShareOut.java")); err != nil {
		t.Fatal(err)
	}
	man, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(man), `android:name="lewkit.FileProvider"`) {
		t.Fatalf("FileProvider:\n%s", man)
	}
}

func TestCreate_FirstPageReplacesSplash(t *testing.T) {
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
	mainJava, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/MainActivity.java"))
	if err != nil {
		t.Fatal(err)
	}
	main := string(mainJava)
	if !strings.Contains(main, "finish()") {
		t.Fatal("splash activity stays on the back stack")
	}
	page, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/PageActivity.java"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(page)
	if !strings.Contains(body, "SwipeRefreshLayout") || !strings.Contains(body, "setOnRefreshListener") {
		t.Fatal("page is missing pull-to-refresh")
	}
	if !strings.Contains(body, "canGoBack()") || !strings.Contains(body, "OnBackInvokedDispatcher") {
		t.Fatal("back leaves the page without walking its history")
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
