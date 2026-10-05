package mac

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBundle_GoExecutable(t *testing.T) {
	work := t.TempDir()
	err := Create(t.Context(), Options{
		OutDir: work,
		Config: Config{
			PackageID: "br.tec.lew.counter",
			AppName:   "Counter",
			GoMain:    ".",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(work, "bin", HelperName)
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("go-bin"), 0o644); err != nil {
		t.Fatal(err)
	}
	icns := filepath.Join(work, "Resources", "AppIcon.icns")
	if err := os.MkdirAll(filepath.Dir(icns), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(icns, []byte("icns"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "Counter.app")
	if err := os.WriteFile(out, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeBundle(out, "Counter", filepath.Join(work, "Info.plist"), bin, icns); err != nil {
		t.Fatal(err)
	}

	exe := filepath.Join(out, "Contents", "MacOS", "Counter")
	body, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "go-bin" {
		t.Fatalf("executable %q", body)
	}
	st, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o111 == 0 {
		t.Fatalf("executable mode %v", st.Mode())
	}
	entries, err := os.ReadDir(filepath.Join(out, "Contents", "MacOS"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "Counter" {
		names := make([]string, len(entries))
		for i, entry := range entries {
			names[i] = entry.Name()
		}
		t.Fatalf("MacOS entries %q", names)
	}
	if _, err := os.Stat(filepath.Join(out, "Contents", "MacOS", HelperName)); !os.IsNotExist(err) {
		t.Fatalf("helper present: %v", err)
	}
	plist, err := os.ReadFile(filepath.Join(out, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := plistValue(string(plist), "CFBundleExecutable")
	if !ok || got != "Counter" {
		t.Fatalf("CFBundleExecutable %q\n%s", got, plist)
	}
	pkg, err := os.ReadFile(filepath.Join(out, "Contents", "PkgInfo"))
	if err != nil {
		t.Fatal(err)
	}
	if string(pkg) != "APPL????" {
		t.Fatalf("PkgInfo %q", pkg)
	}
	icon, err := os.ReadFile(filepath.Join(out, "Contents", "Resources", "AppIcon.icns"))
	if err != nil {
		t.Fatal(err)
	}
	if string(icon) != "icns" {
		t.Fatalf("icon %q", icon)
	}
}

func plistValue(plist, key string) (string, bool) {
	i := strings.Index(plist, "<key>"+key+"</key>")
	if i < 0 {
		return "", false
	}
	rest := plist[i:]
	open := strings.Index(rest, "<string>")
	close := strings.Index(rest, "</string>")
	if open < 0 || close < open {
		return "", false
	}
	return rest[open+len("<string>") : close], true
}
