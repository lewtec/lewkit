package mac

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/build/version"
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

// Build scaffolds the Mac host, cross-compiles the Go helper, and (unless
// GoOnly) runs xcodegen + xcodebuild Debug and copies the .app to OutApp.
func Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = log.Writer()
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = log.Writer()
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
	if _, err := fmt.Fprintf(stdout, "eletrocromo: version %s (code %d)\n", cfg.VersionName, cfg.VersionCode); err != nil {
		return nil, err
	}

	if !opts.GoOnly && runtime.GOOS != "darwin" {
		return nil, ErrMacOSRequired
	}

	workDir, ephemeral, err := common.ResolveWorkDir(opts.WorkDir, "eletrocromo-macos-*")
	if err != nil {
		return nil, err
	}
	var buildErr error
	defer func() {
		if ephemeral && buildErr == nil && !opts.KeepWorkDir && !opts.GoOnly {
			if err := os.RemoveAll(workDir); err != nil && stderr != nil {
				fmt.Fprintf(stderr, "eletrocromo: cleanup work dir: %v\n", err)
			}
		}
	}()

	genCfg := cfg
	genCfg.GoMain = goMain

	if _, err := fmt.Fprintf(stdout, "eletrocromo: generating macOS host in %s\n", workDir); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := Create(Options{OutDir: workDir, Force: true, Config: genCfg}); err != nil {
		buildErr = fmt.Errorf("generate host: %w", err)
		return nil, buildErr
	}

	iconRoot := strings.TrimSpace(opts.IconRoot)
	if iconRoot == "" {
		tmpIcons := filepath.Join(workDir, ".eletrocromo-icons")
		if _, err := icons.Generate(icons.Options{OutputDir: tmpIcons, Force: true}); err != nil {
			buildErr = fmt.Errorf("default icons: %w", err)
			return nil, buildErr
		}
		iconRoot = tmpIcons
	}
	icnsDest := filepath.Join(workDir, "Resources", "AppIcon.icns")
	if _, err := fmt.Fprintf(stdout, "eletrocromo: applying macos icon from %s\n", iconRoot); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := icons.ApplyMacOSICNS(iconRoot, icnsDest); err != nil {
		buildErr = err
		return nil, buildErr
	}

	arch, xArch, err := hostDarwinArch()
	if opts.GOARCH != "" {
		arch, xArch, err = darwinArch(opts.GOARCH)
	}
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	helperDest := filepath.Join(workDir, "bin", HelperName)
	if _, err := fmt.Fprintf(stdout, "eletrocromo: building Go helper (GOOS=darwin GOARCH=%s)\n", arch); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := buildGoHelper(ctx, helperDest, goMain, arch, vi, cfg.PackageID); err != nil {
		buildErr = err
		return nil, buildErr
	}

	result := &BuildResult{WorkDir: workDir, HelperPath: helperDest}
	if opts.GoOnly {
		_, err := fmt.Fprintf(stdout, "eletrocromo: --go-only: skipped xcodebuild (helper %s)\n", helperDest)
		return result, err
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

	if _, err := fmt.Fprintf(stdout, "eletrocromo: assembling Debug .app…\n"); err != nil {
		buildErr = err
		return nil, buildErr
	}
	built, err := assembleDebug(workDir, cfg.ProductName(), xArch, stdout, stderr)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := installHelper(built, helperDest); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := installICNS(built, icnsDest); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := os.MkdirAll(filepath.Dir(outApp), 0o755); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := common.ReplaceDir(built, outApp); err != nil {
		buildErr = fmt.Errorf("copy app: %w", err)
		return nil, buildErr
	}
	result.AppPath = outApp
	_, err = fmt.Fprintf(stdout, "eletrocromo: app → %s\n", outApp)
	return result, err
}

func darwinArch(goarch string) (string, string, error) {
	switch goarch {
	case "arm64":
		return "arm64", "arm64", nil
	case "amd64":
		return "amd64", "x86_64", nil
	default:
		return "", "", fmt.Errorf("%w: %s", ErrUnsupportedGOARCH, goarch)
	}
}

func hostDarwinArch() (goarch, xcodeArch string, err error) {
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

func assembleDebug(workDir, product, xArch string, stdout, stderr io.Writer) (string, error) {
	if _, err := exec.LookPath("xcodegen"); err != nil {
		return "", fmt.Errorf("%w: %w", ErrXcodeGenNotFound, err)
	}
	if _, err := exec.LookPath("xcodebuild"); err != nil {
		return "", fmt.Errorf("%w: %w", ErrXcodebuildNotFound, err)
	}

	gen := exec.Command("xcodegen", "generate")
	gen.Dir = workDir
	gen.Stdout = stdout
	gen.Stderr = stderr
	if err := gen.Run(); err != nil {
		return "", fmt.Errorf("xcodegen generate: %w", err)
	}

	derived := filepath.Join(workDir, "build", "DerivedData")
	proj := filepath.Join(workDir, product+".xcodeproj")
	cmd := exec.Command("xcodebuild",
		"-project", proj,
		"-scheme", product,
		"-configuration", "Debug",
		"-destination", "platform=macOS,arch="+xArch,
		"-derivedDataPath", derived,
		"-skipPackagePluginValidation",
		"CODE_SIGN_IDENTITY=-",
		"CODE_SIGNING_REQUIRED=NO",
		"ENABLE_APP_SANDBOX=NO",
		"ENABLE_DEBUG_DYLIB=NO",
		"ARCHS="+xArch,
		"ONLY_ACTIVE_ARCH=YES",
		"build",
	)
	cmd.Dir = workDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("xcodebuild: %w\n(work dir left at %s)", err, workDir)
	}

	app := filepath.Join(derived, "Build", "Products", "Debug", product+".app")
	if st, err := os.Stat(app); err != nil || !st.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrDebugAppMissing, app)
	}
	return app, nil
}

func installHelper(appPath, helper string) error {
	dest := filepath.Join(appPath, "Contents", "MacOS", HelperName)
	if err := copyFile(helper, dest); err != nil {
		return fmt.Errorf("install helper: %w", err)
	}
	return os.Chmod(dest, 0o755)
}

func installICNS(appPath, icns string) error {
	dest := filepath.Join(appPath, "Contents", "Resources", "AppIcon.icns")
	if err := copyFile(icns, dest); err != nil {
		return fmt.Errorf("install icns: %w", err)
	}
	return resignApp(appPath)
}

func resignApp(appPath string) error {
	// Re-sign after adding the Go child and icon (ad-hoc, like rterm).
	if _, err := exec.LookPath("codesign"); err != nil {
		return nil
	}
	cmd := exec.Command("codesign", "--force", "--deep", "--sign", "-", appPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("codesign: %w (%s)", err, strings.TrimSpace(string(out)))
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
