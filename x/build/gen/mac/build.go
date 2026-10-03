package mac

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/build/version"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Build / toolchain sentinels.
var (
	ErrOutAppRequired     = errors.New("out .app path is required (or use --go-only)")
	ErrMacOSRequired      = errors.New("full .app requires macOS with Xcode; use --go-only")
	ErrXcodebuildNotFound = errors.New("xcodebuild not found; install Xcode")
	ErrXcodeGenNotFound   = errors.New("xcodegen not found; install xcodegen (mise/brew)")
	ErrDebugAppMissing    = errors.New("xcodebuild succeeded but no Debug .app under DerivedData")
	ErrUnsupportedGOARCH  = errors.New("unsupported host GOARCH for darwin")
)

// BuildOptions drives a macos .app build from an eletrocromo app.
type BuildOptions struct {
	Config      Config
	BaseDir     string
	WorkDir     string
	KeepWorkDir bool
	OutApp      string
	GoOnly      bool
	GOARCH      string
	IconRoot    string
	Stdout      io.Writer
	Stderr      io.Writer
}

// BuildResult is the outcome of Build.
type BuildResult struct {
	AppPath    string
	WorkDir    string
	HelperPath string
}

// Build compiles the Go program and, unless GoOnly, writes an unsigned .app
// whose executable is that program.
func Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = taskgroup.LineWriterFrom(ctx)
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = stdout
	}

	cfg, err := opts.Config.withDefaults()
	if err != nil {
		return nil, err
	}
	baseDir := strings.TrimSpace(opts.BaseDir)
	if baseDir == "" {
		baseDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	goMain, err := ResolveGoMain(cfg.GoMain, baseDir)
	if err != nil {
		return nil, err
	}

	vi, name, code := common.StampPackagingVersion(goMain, opts.Config.VersionName, opts.Config.VersionCode)
	cfg.VersionName = name
	cfg.VersionCode = code
	slog.Info("macos version", "version", cfg.VersionName, "code", cfg.VersionCode)

	workDir, ephemeral, err := common.ResolveWorkDir(opts.WorkDir, "eletrocromo-macos-*")
	if err != nil {
		return nil, err
	}
	var buildErr error
	defer func() {
		if ephemeral && buildErr == nil && !opts.KeepWorkDir && !opts.GoOnly {
			if err := os.RemoveAll(workDir); err != nil && stderr != nil {
				slog.Warn("macos cleanup", "err", err)
			}
		}
	}()

	genCfg := cfg
	genCfg.GoMain = goMain

	slog.Info("macos host", "dir", workDir)
	if err := Create(Options{OutDir: workDir, Force: true, Config: genCfg}); err != nil {
		buildErr = fmt.Errorf("generate host: %w", err)
		return nil, buildErr
	}

	iconRoot := strings.TrimSpace(opts.IconRoot)
	if iconRoot == "" {
		tmpIcons := filepath.Join(workDir, ".eletrocromo-icons")
		if _, err := icons.Generate(icons.Options{OutputDir: tmpIcons, Force: true, SourcePath: icons.MasterPath(baseDir, cfg.Icon)}); err != nil {
			buildErr = fmt.Errorf("icons: %w", err)
			return nil, buildErr
		}
		iconRoot = tmpIcons
	}
	icnsDest := filepath.Join(workDir, "Resources", "AppIcon.icns")
	slog.Info("macos icon", "dir", iconRoot)
	if err := icons.ApplyMacOSICNS(iconRoot, icnsDest); err != nil {
		buildErr = err
		return nil, buildErr
	}

	arch, err := hostDarwinArch()
	if opts.GOARCH != "" {
		arch, err = darwinArch(opts.GOARCH)
	}
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	helperDest := filepath.Join(workDir, "bin", HelperName)
	slog.Info("macos go", "arch", arch)
	if err := buildGoHelper(ctx, helperDest, goMain, arch, vi, cfg.PackageID); err != nil {
		buildErr = err
		return nil, buildErr
	}

	result := &BuildResult{WorkDir: workDir, HelperPath: helperDest}
	if opts.GoOnly {
		slog.Info("macos go-only", "helper", helperDest)
		return result, nil
	}

	outApp := strings.TrimSpace(opts.OutApp)
	if outApp == "" {
		buildErr = ErrOutAppRequired
		return nil, buildErr
	}
	outApp, err = filepath.Abs(outApp)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}

	slog.Info("macos bundle", "path", outApp)
	if err := writeBundle(outApp, cfg.ProductName(), filepath.Join(workDir, "Info.plist"), helperDest, icnsDest); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if runtime.GOOS == "darwin" {
		if err := resignApp(ctx, outApp); err != nil {
			buildErr = err
			return nil, buildErr
		}
	}
	result.AppPath = outApp
	slog.Info("macos app", "path", outApp)
	return result, nil
}

func writeBundle(outApp, product, plistPath, binaryPath, icnsPath string) error {
	if err := os.RemoveAll(outApp); err != nil {
		return err
	}
	macOS := filepath.Join(outApp, "Contents", "MacOS")
	resources := filepath.Join(outApp, "Contents", "Resources")
	if err := os.MkdirAll(macOS, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(resources, 0o755); err != nil {
		return err
	}
	if err := copyFile(plistPath, filepath.Join(outApp, "Contents", "Info.plist")); err != nil {
		return fmt.Errorf("bundle plist: %w", err)
	}
	exe := filepath.Join(macOS, product)
	if err := copyFile(binaryPath, exe); err != nil {
		return fmt.Errorf("bundle executable: %w", err)
	}
	if err := os.Chmod(exe, 0o755); err != nil {
		return err
	}
	if icnsPath != "" {
		if err := copyFile(icnsPath, filepath.Join(resources, "AppIcon.icns")); err != nil {
			return fmt.Errorf("bundle icon: %w", err)
		}
	}
	if err := os.WriteFile(filepath.Join(outApp, "Contents", "PkgInfo"), []byte("APPL????"), 0o644); err != nil {
		return err
	}
	return nil
}

func darwinArch(goarch string) (string, error) {
	switch goarch {
	case "arm64", "amd64":
		return goarch, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedGOARCH, goarch)
	}
}

func hostDarwinArch() (string, error) {
	return darwinArch(runtime.GOARCH)
}

func buildGoHelper(ctx context.Context, dest, goMainDir, goarch string, stamp version.Info, appID string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	err := gocmd.Command{
		Dir: goMainDir,
		Env: append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS=darwin",
			"GOARCH="+goarch,
		),
		Args: []string{"-trimpath", "-ldflags", stamp.WithAppID(appID), "-o", dest, "."},
	}.Run(ctx)
	if err != nil {
		return fmt.Errorf("go build darwin/%s: %w", goarch, err)
	}
	return os.Chmod(dest, 0o755)
}

func resignApp(ctx context.Context, appPath string) error {
	if _, err := execdriver.Which(ctx, "codesign"); err != nil {
		return nil
	}
	if err := gocmd.Tool(ctx, "codesign", "", nil, "--force", "--deep", "--sign", "-", appPath); err != nil {
		return fmt.Errorf("codesign: %w", err)
	}
	return nil
}

// macIconSlots are the AppIcon.appiconset filenames from the host template.
var macIconSlots = []struct {
	name string
	px   int
}{
	{"icon_16x16.png", 16},
	{"icon_16x16@2x.png", 32},
	{"icon_32x32.png", 32},
	{"icon_32x32@2x.png", 64},
	{"icon_128x128.png", 128},
	{"icon_128x128@2x.png", 256},
	{"icon_256x256.png", 256},
	{"icon_256x256@2x.png", 512},
	{"icon_512x512.png", 512},
	{"icon_512x512@2x.png", 1024},
}

func applyMacIcons(iconRoot, assetsDir string) error {
	img, err := icons.DecodeImage(filepath.Join(iconRoot, "source", "master.png"))
	if err != nil {
		return fmt.Errorf("macos app icon: %w", err)
	}
	if img.Bounds().Dx() < 1 || img.Bounds().Dy() < 1 {
		return errors.New("macos app icon: empty master")
	}
	iconDir := filepath.Join(assetsDir, "AppIcon.appiconset")
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return err
	}
	for _, slot := range macIconSlots {
		if err := icons.WritePNG(filepath.Join(iconDir, slot.name), icons.Resize(img, slot.px)); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return errors.Join(err, in.Close())
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return errors.Join(err, in.Close())
	}
	_, copyErr := io.Copy(out, in)
	return errors.Join(copyErr, out.Close(), in.Close())
}
