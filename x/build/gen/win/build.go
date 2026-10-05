package win

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
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Build / toolchain sentinels.
var (
	ErrOutExeRequired    = errors.New("out .exe path is required (or use --go-only)")
	ErrUnsupportedGOARCH = errors.New("unsupported GOARCH for windows")
)

// BuildOptions drives a windows GUI .exe build from an eletrocromo app.
type BuildOptions struct {
	Config      Config
	BaseDir     string
	WorkDir     string
	KeepWorkDir bool
	OutExe      string
	GoOnly      bool
	GOARCH      string
	IconRoot    string
	Stdout      io.Writer
	Stderr      io.Writer
}

// BuildResult is the outcome of Build.
type BuildResult struct {
	ExePath string
	WorkDir string
}

// Build compiles the Go program as a GUI executable and stamps the icon,
// manifest, and version info. GoOnly stops after that executable, in WorkDir.
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
	arch, err := windowsArch(opts.GOARCH)
	if err != nil {
		return nil, err
	}

	vi, name, code := common.StampPackagingVersion(ctx, goMain, opts.Config.VersionName, opts.Config.VersionCode)
	cfg.VersionName = name
	cfg.VersionCode = code
	slog.Info("windows version", "version", cfg.VersionName, "code", cfg.VersionCode)

	workDir, ephemeral, err := common.ResolveWorkDir(opts.WorkDir, release.Name()+"-windows-*")
	if err != nil {
		return nil, err
	}
	var buildErr error
	defer func() {
		if ephemeral && buildErr == nil && !opts.KeepWorkDir && !opts.GoOnly {
			if err := os.RemoveAll(workDir); err != nil && stderr != nil {
				slog.Warn("windows cleanup", "err", err)
			}
		}
	}()

	iconRoot := strings.TrimSpace(opts.IconRoot)
	if iconRoot == "" {
		tmpIcons := filepath.Join(workDir, "icons")
		if _, err := icons.Generate(icons.Options{OutputDir: tmpIcons, Force: true, SourcePath: icons.MasterPath(baseDir, cfg.Icon)}); err != nil {
			buildErr = fmt.Errorf("icons: %w", err)
			return nil, buildErr
		}
		iconRoot = tmpIcons
	}
	icoPath := filepath.Join(iconRoot, "windows", "icon.ico")

	product := cfg.ProductName()
	built := filepath.Join(workDir, product+".exe")
	slog.Info("windows go", "arch", arch)
	if err := buildGo(ctx, built, goMain, arch, vi, cfg.PackageID); err != nil {
		buildErr = err
		return nil, buildErr
	}
	slog.Info("windows resources", "exe", built)
	if err := stampExe(built, icoPath, cfg); err != nil {
		buildErr = err
		return nil, buildErr
	}

	result := &BuildResult{WorkDir: workDir, ExePath: built}
	if opts.GoOnly {
		slog.Info("windows go-only", "exe", built)
		return result, nil
	}
	out := strings.TrimSpace(opts.OutExe)
	if out == "" {
		buildErr = ErrOutExeRequired
		return nil, buildErr
	}
	out, err = filepath.Abs(out)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := common.CopyFile(built, out, 0o755); err != nil {
		buildErr = fmt.Errorf("copy exe: %w", err)
		return nil, buildErr
	}
	result.ExePath = out
	slog.Info("windows app", "path", out)
	return result, nil
}

func windowsArch(goarch string) (string, error) {
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	switch goarch {
	case "amd64", "arm64":
		return goarch, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedGOARCH, goarch)
	}
}

func buildGo(ctx context.Context, dest, goMainDir, goarch string, stamp version.Info, appID string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	err := gocmd.Command{
		Dir: goMainDir,
		Env: append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS=windows",
			"GOARCH="+goarch,
		),
		Args: []string{"-trimpath", "-ldflags", "-H windowsgui " + stamp.WithAppID(appID), "-o", dest, "."},
	}.Run(ctx)
	if err != nil {
		return fmt.Errorf("go build windows/%s: %w", goarch, err)
	}
	return nil
}
