package apk

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreate_PackageIDLayout(t *testing.T) {
	out := t.TempDir()
	err := Create(t.Context(), Options{
		OutDir: out,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    "../../examples/counter",
		},
	})
	require.NoError(t, err)

	mustExist := []string{
		"eletrocromo.json",
		"settings.gradle",
		"app/build.gradle",
		"app/src/main/AndroidManifest.xml",
		"app/src/main/java/br/tec/lew/counter/MainActivity.java",
		"app/src/main/java/lewkit/FileChooser.java",
		"app/src/main/java/lewkit/Ask.java",
		"app/src/main/java/lewkit/Notify.java",
		"app/src/main/java/lewkit/Open.java",
		"app/src/main/java/lewkit/HostActivity.java",
		"app/src/main/java/lewkit/Page.java",
		"app/src/main/java/lewkit/Documents.java",
		"app/src/main/java/br/tec/lew/counter/PageActivity.java",
		"app/src/main/java/br/tec/lew/counter/Windows.java",
		"app/src/main/res/xml/network_security_config.xml",
		"app/src/main/res/layout/activity_main.xml",
		"scripts/build-go.sh",
		"README.md",
	}
	for _, rel := range mustExist {
		_, err := os.Stat(filepath.Join(out, rel))
		assert.NoError(t, err, "missing %s", rel)
	}

	gradle, err := os.ReadFile(filepath.Join(out, "app/build.gradle"))
	require.NoError(t, err)
	s := string(gradle)
	require.Contains(t, s, `applicationId = "br.tec.lew.counter"`)
	require.Contains(t, s, `namespace = "br.tec.lew.counter"`)
	require.Contains(t, s, "swiperefreshlayout:swiperefreshlayout:1.1.0")
	require.NotContains(t, s, "kotlin-stdlib")
	require.NotContains(t, s, "appcompat")
	require.NotContains(t, s, "androidx.core")
	require.NotContains(t, s, "androidx.webkit")

	mainJava, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/MainActivity.java"))
	require.NoError(t, err)
	main := string(mainJava)
	require.True(t, strings.HasPrefix(main, "package br.tec.lew.counter;\n"), "java package mismatch:\n%s", main[:80])
	require.Contains(t, main, "extends HostActivity")
	require.Contains(t, main, "R.id.splash")
	require.Contains(t, main, "url.isEmpty()")
	require.Contains(t, main, "Host.onPage")
	require.NotContains(t, main, "ServerService")
	require.NotContains(t, main, "startForegroundService")
	manifest, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	require.NoError(t, err)
	man := string(manifest)
	require.NotContains(t, man, "ServerService")
	require.NotContains(t, man, "FOREGROUND_SERVICE")
	require.NotContains(t, main, "android.webkit.WebView")
	require.NotContains(t, main, "webview_container")
	layout, err := os.ReadFile(filepath.Join(out, "app/src/main/res/layout/activity_main.xml"))
	require.NoError(t, err)
	lay := string(layout)
	require.NotContains(t, lay, "webview_container")
	require.NotContains(t, lay, "SwipeRefreshLayout")
	surface, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/SurfaceActivity.java"))
	require.NoError(t, err)
	surf := string(surface)
	require.Contains(t, surf, "extends HostActivity")
	require.Contains(t, surf, "Host.boot(this)")
	require.Contains(t, surf, "setFixedSize")
	require.Contains(t, surf, "Host.resize")
	require.Contains(t, surf, "getCurrentWindowMetrics")
	require.Contains(t, surf, "addOnLayoutChangeListener")
	require.Contains(t, surf, "setDecorFitsSystemWindows(false)")
	require.Contains(t, surf, "LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES")
	require.NotContains(t, surf, "SYSTEM_UI_FLAG_HIDE_NAVIGATION")
	require.NotContains(t, surf, "SYSTEM_UI_FLAG_FULLSCREEN")
	require.NotContains(t, surf, "IMMERSIVE")

	cfg, err := os.ReadFile(filepath.Join(out, "eletrocromo.json"))
	require.NoError(t, err)
	cfgText := string(cfg)
	require.Contains(t, cfgText, `"package_id": "br.tec.lew.counter"`)
	require.Contains(t, cfgText, `"go_main": "../../examples/counter"`)

	manifest, err = os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	require.NoError(t, err)
	man = string(manifest)
	require.NotContains(t, man, `android:scheme=`)
	require.Contains(t, man, `android:configChanges="orientation|screenSize|keyboardHidden|uiMode|fontScale"`)
	mainAt := strings.Index(man, `android:name=".MainActivity"`)
	launchAt := strings.Index(man, "LAUNCHER")
	surfaceAt := strings.Index(man, `android:name="lewkit.SurfaceActivity"`)
	require.GreaterOrEqual(t, mainAt, 0, "web launcher:\n%s", man)
	require.GreaterOrEqual(t, launchAt, mainAt, "web launcher:\n%s", man)
	require.GreaterOrEqual(t, surfaceAt, launchAt, "web launcher:\n%s", man)
	require.NotContains(t, man, `android:name="lewkit.FileChooser"`)
	chooser, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/FileChooser.java"))
	require.NoError(t, err)
	chooserJava := string(chooser)
	require.NotContains(t, chooserJava, "extends Activity")
	require.NotContains(t, chooserJava, "FileChooser.class")
	require.Contains(t, chooserJava, "Host.foreground")
	require.Contains(t, chooserJava, "startActivityForResult")
	require.Contains(t, chooserJava, "ACTION_OPEN_DOCUMENT_TREE")
	require.Contains(t, chooserJava, "ACTION_OPEN_DOCUMENT")
	require.Contains(t, chooserJava, "ACTION_CREATE_DOCUMENT")
	hostAct, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/HostActivity.java"))
	require.NoError(t, err)
	hostActJava := string(hostAct)
	hostJava, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Host.java"))
	require.NoError(t, err)
	host := string(hostJava)
	require.Contains(t, host, "AlertDialog")
	require.Contains(t, host, "showFail")
	askJava, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Ask.java"))
	require.NoError(t, err)
	askText := string(askJava)
	require.Contains(t, askText, "void alert")
	require.Contains(t, askText, "void confirm")
	require.Contains(t, askText, "void prompt")
	require.Contains(t, askText, "void choose")
	require.Contains(t, hostActJava, "Host.noteForeground(this, true)")
	require.Contains(t, hostActJava, "FileChooser.onResult")
	require.Contains(t, hostActJava, "FileChooser.hostGone")
	require.Contains(t, hostActJava, "Notify.onResult")
	require.Contains(t, man, "android.permission.POST_NOTIFICATIONS")
	note, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Notify.java"))
	require.NoError(t, err)
	noteJava := string(note)
	require.Contains(t, noteJava, "NotificationManager")
	require.Contains(t, noteJava, "createNotificationChannel")
	require.Contains(t, noteJava, "requestPermissions")
	openJava, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Open.java"))
	require.NoError(t, err)
	openText := string(openJava)
	require.Contains(t, openText, "ACTION_VIEW")
	require.Contains(t, openText, "FileProvider.getUriForFile")
	require.Contains(t, openText, "startActivity")
	pageJava, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/PageActivity.java"))
	require.NoError(t, err)
	page := string(pageJava)
	require.Contains(t, page, "extends HostActivity")
	require.Contains(t, page, "wv.onResume()")
	require.NotContains(t, main, "onActivityResult")
	require.NotContains(t, page, "onActivityResult")
	require.NotContains(t, surf, "onActivityResult")
	docs, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Documents.java"))
	require.NoError(t, err)
	docText := string(docs)
	require.Contains(t, docText, "int readFd")
	require.Contains(t, docText, "int thumbFd")

	hostJava, err = os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Host.java"))
	require.NoError(t, err)
	host = string(hostJava)
	require.NotContains(t, host, "dataDir")
	require.NotContains(t, host, "cacheDir")
	require.NotContains(t, host, "configDir")
	require.Contains(t, host, `cb.call("")`)
	splashAt := strings.Index(host, "Ready splash = onReady")
	pageAt := strings.Index(host, "Ready page = onPage")
	require.GreaterOrEqual(t, splashAt, 0)
	require.GreaterOrEqual(t, pageAt, splashAt)
	require.NoError(t, filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".kt") || strings.HasSuffix(path, ".kts") {
			assert.Failf(t, "kotlin source", "%s", path)
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
			assert.NotContains(t, text, banned, path)
		}
		return nil
	}))

	proxy, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/GoProxy.java"))
	require.NoError(t, err)
	require.Contains(t, string(proxy), "nativeInvoke")

	sh, err := os.ReadFile(filepath.Join(out, "scripts/build-go.sh"))
	require.NoError(t, err)
	info, err := os.Stat(filepath.Join(out, "scripts/build-go.sh"))
	require.NoError(t, err)
	require.NotZero(t, info.Mode()&0o111, "build-go.sh not executable: %v", info.Mode())
	script := string(sh)
	require.Contains(t, script, "GOOS=android")
	require.NotContains(t, script, "androidnocgo")
	require.Contains(t, script, "-checklinkname=0")
	require.Contains(t, script, "androidtoolexec")
	require.Contains(t, script, "_cgo_android_export.go")
	require.Contains(t, script, "-tags appmode")
}

func TestCreate_CapabilitiesIntentFilters(t *testing.T) {
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
	manifest, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	require.NoError(t, err)
	s := string(manifest)
	require.Contains(t, s, `android:scheme="myapp"`)
	require.Contains(t, s, `android:mimeType="text/markdown"`)
	_, err = os.Stat(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/OpenDrop.java"))
	require.NoError(t, err)
	shareOut, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/ShareOut.java"))
	require.NoError(t, err)
	require.Contains(t, string(shareOut), "existing.length()")
	man, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	require.NoError(t, err)
	require.Contains(t, string(man), `android:name="lewkit.FileProvider"`)
}

func TestCreate_FirstPageReplacesSplash(t *testing.T) {
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
	mainJava, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/MainActivity.java"))
	require.NoError(t, err)
	main := string(mainJava)
	require.Contains(t, main, "finish()")
	require.NotContains(t, main, "WebView")
	page, err := os.ReadFile(filepath.Join(out, "app/src/main/java/br/tec/lew/counter/PageActivity.java"))
	require.NoError(t, err)
	body := string(page)
	require.Contains(t, body, "SwipeRefreshLayout")
	require.Contains(t, body, "setOnRefreshListener")
	require.Contains(t, body, "canGoBack()")
	require.Contains(t, body, "OnBackInvokedDispatcher")
	require.Contains(t, body, "EXTRA_PAGE")
	require.Contains(t, body, "Page.intercept")
	require.Contains(t, body, "lewkitPage")
	require.Contains(t, main, "Page.onOpen")
	require.LessOrEqual(t, strings.Index(main, "Page.onOpen"), strings.Index(main, "Host.boot"))
	pageHost, err := os.ReadFile(filepath.Join(out, "app/src/main/java/lewkit/Page.java"))
	require.NoError(t, err)
	pageText := string(pageHost)
	require.True(t, strings.Contains(pageText, "shouldInterceptRequest") || strings.Contains(pageText, "WebResourceRequest"))
	require.NotContains(t, pageText, "ServerService")
	require.NotContains(t, pageText, "127.0.0.1")
	require.Contains(t, pageText, `Host.ready("")`)
	manifest, err := os.ReadFile(filepath.Join(out, "app/src/main/AndroidManifest.xml"))
	require.NoError(t, err)
	man := string(manifest)
	require.Contains(t, man, `android:name=".PageActivity"`)
	require.Contains(t, man, `android:launchMode="standard"`)
}

func TestCreate_RejectsBadID(t *testing.T) {
	err := Create(t.Context(), Options{
		OutDir: t.TempDir(),
		Config: Config{PackageID: "Not.Valid"},
	})
	require.Error(t, err)
}

func TestCreate_RequiresForceWhenNonEmpty(t *testing.T) {
	out := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(out, "keep"), []byte("x"), 0o644))
	err := Create(t.Context(), Options{
		OutDir: out,
		Config: Config{PackageID: "br.tec.lew.x"},
	})
	require.Error(t, err)
	require.NoError(t, Create(t.Context(), Options{
		OutDir: out,
		Force:  true,
		Config: Config{PackageID: "br.tec.lew.x", AppName: "X"},
	}))
	_, err = os.Stat(filepath.Join(out, "keep"))
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestCreate_DefaultAppNameFromID(t *testing.T) {
	out := t.TempDir()
	require.NoError(t, Create(t.Context(), Options{
		OutDir: out,
		Config: Config{PackageID: "br.tec.lew.myapp"},
	}))
	stringsXML, err := os.ReadFile(filepath.Join(out, "app/src/main/res/values/strings.xml"))
	require.NoError(t, err)
	require.Contains(t, string(stringsXML), ">myapp</string>")
}
