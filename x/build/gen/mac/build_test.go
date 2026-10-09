package mac

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallHelper(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Counter.app")
	macOS := filepath.Join(app, "Contents", "MacOS")
	if err := os.MkdirAll(macOS, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(macOS, "Counter"), []byte("swift-shell"), 0o755); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(t.TempDir(), HelperName)
	if err := os.WriteFile(helper, []byte("go-server"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installHelper(app, helper); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(macOS, HelperName))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "go-server" {
		t.Fatalf("server %q", body)
	}
	shell, err := os.ReadFile(filepath.Join(macOS, "Counter"))
	if err != nil {
		t.Fatal(err)
	}
	if string(shell) != "swift-shell" {
		t.Fatalf("shell %q", shell)
	}
	st, err := os.Stat(filepath.Join(macOS, HelperName))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o111 == 0 {
		t.Fatalf("server mode %v", st.Mode())
	}
}

func TestBuildFullAppNeedsMacOS(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("full build uses xcodebuild")
	}
	mainDir := t.TempDir()
	_, err := Build(t.Context(), BuildOptions{
		Config: Config{
			PackageID:   "br.tec.lew.demo",
			AppName:     "Demo",
			VersionName: "1.2.3",
			VersionCode: 5,
			GoMain:      mainDir,
		},
		BaseDir: mainDir,
		WorkDir: t.TempDir(),
		OutApp:  filepath.Join(t.TempDir(), "Demo.app"),
	})
	if !errors.Is(err, ErrMacOSRequired) {
		t.Fatalf("err %v", err)
	}
}
