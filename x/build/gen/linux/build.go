package linux

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/build/version"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
)

var (
	ErrGoBuildLinux              = errors.New("go build linux")
	ErrIcon256pxLinuxIconMissing = errs.Wrap(errs.ErrMissing, "icon: 256px linux icon missing")
	ErrLinuxIconIcon             = errors.New("icon")
)

// Build / toolchain sentinels.
var (
	ErrOutAppRequired    = errs.Wrap(errs.ErrRequired, "out .AppImage path is required (or use --go-only)")
	ErrUnsupportedGOARCH = errs.Wrap(errs.ErrUnsupported, "unsupported GOARCH for linux")
)

// BuildOptions drives a Linux desktop app build from an eletrocromo app.
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
	AppPath string
	WorkDir string
}

// Build compiles the Go program and writes one AppImage file.
// GoOnly leaves that file in WorkDir.
func Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = taskgroup.LineWriterFrom(ctx)
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = stdout
	}
	_ = stdout

	cfg, err := opts.Config.withDefaults(ctx)
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
	arch, err := linuxArch(opts.GOARCH)
	if err != nil {
		return nil, err
	}

	vi, name, code := common.StampPackagingVersion(ctx, goMain, opts.Config.VersionName, opts.Config.VersionCode)
	cfg.VersionName = name
	cfg.VersionCode = code
	slog.Info("linux version", "version", cfg.VersionName, "code", cfg.VersionCode)

	workDir, ephemeral, err := common.ResolveWorkDir(opts.WorkDir, release.Name()+"-linux-*")
	if err != nil {
		return nil, err
	}
	var buildErr error
	defer func() {
		if ephemeral && buildErr == nil && !opts.KeepWorkDir && !opts.GoOnly {
			if err := os.RemoveAll(workDir); err != nil && stderr != nil {
				slog.Warn("linux cleanup", "err", err)
			}
		}
	}()

	iconRoot := strings.TrimSpace(opts.IconRoot)
	if iconRoot == "" {
		tmpIcons := filepath.Join(workDir, ".icons")
		if _, err := icons.Generate(icons.Options{OutputDir: tmpIcons, Force: true, SourcePath: icons.MasterPath(baseDir, cfg.Icon)}); err != nil {
			buildErr = fmt.Errorf("icons: %w", err)
			return nil, buildErr
		}
		iconRoot = tmpIcons
	}

	product := cfg.ProductName()
	built := filepath.Join(workDir, product)
	slog.Info("linux go", "arch", arch)
	if err := buildGo(ctx, built, goMain, arch, vi, cfg.PackageID); err != nil {
		buildErr = err
		return nil, buildErr
	}
	icon, err := linuxIcon(iconRoot)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := os.RemoveAll(filepath.Join(workDir, ".icons")); err != nil {
		buildErr = err
		return nil, buildErr
	}
	image := filepath.Join(workDir, product+".AppImage")
	if err := os.Rename(built, image); err != nil {
		buildErr = err
		return nil, buildErr
	}
	desktop := DesktopFile(cfg.AppName, cfg.PackageID, cfg.VersionName, product, "icon")
	if err := release.AppendTrailer(image, map[string][]byte{
		release.MarkerFile:   []byte(cfg.PackageID + "\n"),
		"icon.png":           icon,
		product + ".desktop": []byte(desktop),
	}); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := os.Chmod(image, 0o755); err != nil {
		buildErr = err
		return nil, buildErr
	}

	result := &BuildResult{WorkDir: workDir, AppPath: image}
	if opts.GoOnly {
		slog.Info("linux go-only", "app", image)
		return result, nil
	}
	out := strings.TrimSpace(opts.OutApp)
	if out == "" {
		buildErr = ErrOutAppRequired
		return nil, buildErr
	}
	out, err = filepath.Abs(out)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := publishFile(image, out); err != nil {
		buildErr = fmt.Errorf("copy app: %w", err)
		return nil, buildErr
	}
	result.AppPath = out
	slog.Info("linux app", "path", out)
	return result, nil
}

func linuxIcon(iconRoot string) ([]byte, error) {
	raw, err := os.ReadFile(filepath.Join(iconRoot, "linux", "icon-256.png"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLinuxIconIcon, err)
	}
	if len(raw) == 0 {
		return nil, ErrIcon256pxLinuxIconMissing
	}
	return raw, nil
}

func publishFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(dst); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return os.Chmod(dst, 0o755)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Remove(src)
}

func linuxArch(goarch string) (string, error) {
	return common.Arch(goarch, ErrUnsupportedGOARCH)
}

func buildGo(ctx context.Context, dest, goMainDir, goarch string, stamp version.Info, appID string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	err := gocmd.Command{
		Dir: goMainDir,
		Env: append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS=linux",
			"GOARCH="+goarch,
		),
		Args: []string{"-trimpath", "-ldflags", stamp.WithAppID(appID), "-o", dest, "."},
	}.Run(ctx)
	if err != nil {
		return fmt.Errorf("%w/%s: %w", ErrGoBuildLinux, goarch, err)
	}
	return nil
}
