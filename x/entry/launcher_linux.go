//go:build linux

package entry

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/release"
)

func dataHome() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share"), nil
}

// publishLauncher copies the bundle desktop entry and hicolor icons into the
// user data directory so the session shell can show the app icon.
func publishLauncher(exe, dataDir string) error {
	exe = strings.TrimSpace(exe)
	dataDir = strings.TrimSpace(dataDir)
	if exe == "" || dataDir == "" {
		return nil
	}
	dir := filepath.Dir(exe)
	raw, err := os.ReadFile(filepath.Join(dir, release.MarkerFile))
	if err != nil {
		return err
	}
	id := strings.TrimSpace(string(raw))
	if id == "" {
		return fmt.Errorf("linux launcher: empty app id")
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*.desktop"))
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("linux launcher: desktop file missing")
	}
	body, err := os.ReadFile(matches[0])
	if err != nil {
		return err
	}
	icon := filepath.Join(dir, "icon.png")
	text := rewriteDesktop(string(body), exe, icon)
	appDir := filepath.Join(dataDir, "applications")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(appDir, id+".desktop"), []byte(text), 0o644); err != nil {
		return err
	}
	_ = os.Chmod(matches[0], 0o755)
	srcIcons := filepath.Join(dir, "icons", "hicolor")
	if _, err := os.Stat(srcIcons); err != nil {
		return nil
	}
	return copyTree(srcIcons, filepath.Join(dataDir, "icons", "hicolor"))
}

func rewriteDesktop(body, execPath, iconPath string) string {
	var b strings.Builder
	execDone := false
	iconDone := false
	for _, line := range strings.Split(strings.TrimRight(body, "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "Exec="):
			fmt.Fprintf(&b, "Exec=%s\n", quoteExec(execPath))
			execDone = true
		case strings.HasPrefix(line, "Icon="):
			fmt.Fprintf(&b, "Icon=%s\n", desktopField(iconPath))
			iconDone = true
		default:
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	if !execDone {
		fmt.Fprintf(&b, "Exec=%s\n", quoteExec(execPath))
	}
	if !iconDone {
		fmt.Fprintf(&b, "Icon=%s\n", desktopField(iconPath))
	}
	return b.String()
}

func quoteExec(path string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range path {
		switch r {
		case '\\', '"', '`', '$':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	b.WriteByte('"')
	return b.String()
}

func desktopField(s string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\r", `\r`).Replace(s)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
