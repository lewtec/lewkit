package build

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/tc-hib/winres"
)

// Linux builds the CGO-free binary and writes a png and a desktop entry beside it.
// GoOnly skips those icon files.
func (host Host) Linux(ctx context.Context) (string, error) {
	return host.packageDesktop(ctx, "linux")
}

// Windows builds the CGO-free binary and embeds windows/icon.ico in the exe.
// The exe is a GUI program, so opening it does not create a console.
// GoOnly skips the icon.
func (host Host) Windows(ctx context.Context) (string, error) {
	return host.packageDesktop(ctx, "windows")
}

func (host Host) packageDesktop(ctx context.Context, goos string) (string, error) {
	cfg, base, err := host.Load()
	if err != nil {
		return "", err
	}
	arch := host.GOARCH
	if arch == "" {
		arch = runtime.GOARCH
	}
	if err := (goBinary{
		dir:        cfg.GoMain,
		appID:      cfg.PackageID,
		version:    cfg.VersionName,
		goos:       goos,
		goarch:     arch,
		dest:       host.Out,
		windowsGUI: goos == "windows",
	}).compile(ctx); err != nil {
		return "", err
	}
	if goos == "linux" {
		if err := os.Chmod(host.Out, 0o755); err != nil {
			return "", err
		}
	}
	if host.GoOnly {
		return host.Out, nil
	}
	var iconDir string
	cleanup := func() {}
	if strings.TrimSpace(host.Work) != "" {
		iconDir = filepath.Join(host.Work, ".eletrocromo-icons")
		if err := os.MkdirAll(iconDir, 0o755); err != nil {
			return "", err
		}
	} else {
		iconDir, err = os.MkdirTemp("", "lewkit-icons-")
		if err != nil {
			return "", err
		}
		cleanup = func() { os.RemoveAll(iconDir) }
	}
	defer cleanup()
	if _, err := icons.Generate(icons.Options{
		OutputDir:  iconDir,
		Force:      true,
		SourcePath: icons.MasterPath(base, cfg.Icon),
	}); err != nil {
		return "", fmt.Errorf("icons: %w", err)
	}
	slog.Info("icon " + goos)
	switch goos {
	case "windows":
		err = embedWindowsIcon(host.Out, filepath.Join(iconDir, "windows", "icon.ico"))
	case "linux":
		err = writeLinuxDesktop(host.Out, cfg.AppName, iconDir)
	}
	if err != nil {
		return "", err
	}
	return host.Out, nil
}

func embedWindowsIcon(exePath, icoPath string) error {
	icon, err := loadICO(icoPath)
	if err != nil {
		return err
	}
	return rewriteEXE(exePath, func(rs *winres.ResourceSet) error {
		if err := rs.SetIcon(winres.ID(1), icon); err != nil {
			return fmt.Errorf("windows icon: %w", err)
		}
		return nil
	})
}

func rewriteEXE(exePath string, edit func(*winres.ResourceSet) error) error {
	exe, err := os.Open(exePath)
	if err != nil {
		return err
	}
	defer exe.Close()
	rs, err := winres.LoadFromEXE(exe)
	if err != nil && !errors.Is(err, winres.ErrNoResources) {
		return err
	}
	if err := edit(rs); err != nil {
		return err
	}
	if _, err := exe.Seek(0, io.SeekStart); err != nil {
		return err
	}
	info, err := exe.Stat()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(exePath), ".lewkit-icon-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	remove := true
	defer func() {
		if remove {
			os.Remove(tmpName)
		}
	}()
	if err := rs.WriteToEXE(tmp, exe); err != nil {
		tmp.Close()
		return fmt.Errorf("windows resources: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := exe.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, info.Mode()); err != nil {
		return err
	}
	if err := os.Rename(tmpName, exePath); err != nil {
		return err
	}
	remove = false
	return nil
}

func loadICO(path string) (*winres.Icon, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	icon, err := winres.LoadICO(file)
	if err != nil {
		return nil, fmt.Errorf("windows icon: %w", err)
	}
	return icon, nil
}

func writeLinuxDesktop(binPath, appName, iconRoot string) error {
	src, err := linuxPNG(iconRoot)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	dir := filepath.Dir(binPath)
	base := filepath.Base(binPath)
	pngPath := filepath.Join(dir, base+".png")
	if err := os.WriteFile(pngPath, raw, 0o644); err != nil {
		return err
	}
	binAbs, err := filepath.Abs(binPath)
	if err != nil {
		return err
	}
	pngAbs, err := filepath.Abs(pngPath)
	if err != nil {
		return err
	}
	body := "[Desktop Entry]\nType=Application\nName=" + desktopName(appName, binPath) + "\nExec=" + desktopExec(binAbs) + "\nIcon=" + pngAbs + "\nTerminal=false\n"
	return os.WriteFile(filepath.Join(dir, base+".desktop"), []byte(body), 0o644)
}

func linuxPNG(iconRoot string) (string, error) {
	sizes := icons.LinuxPNGSizes
	for i := len(sizes) - 1; i >= 0; i-- {
		name := filepath.Join(iconRoot, "linux", fmt.Sprintf("icon-%d.png", sizes[i]))
		info, err := os.Stat(name)
		if err == nil && !info.IsDir() {
			return name, nil
		}
	}
	return "", fmt.Errorf("linux icon png missing under %s", iconRoot)
}

func desktopName(appName, binPath string) string {
	name := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' {
			return ' '
		}
		return r
	}, appName)
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return filepath.Base(binPath)
	}
	return name
}

func desktopExec(path string) string {
	if strings.ContainsAny(path, " \t\"\\") {
		return strconv.Quote(path)
	}
	return path
}
