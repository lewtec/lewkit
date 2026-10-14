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
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/image/convert"
)

var (
	ErrBundleServer            = errors.New("bundle server")
	ErrGoBuildDarwin           = errors.New("go build darwin")
	ErrMacosAppIcon            = errors.New("macos app icon")
	ErrMacosAppIconEmptyMaster = errs.Wrap(errs.ErrEmpty, "macos app icon: empty master")
	ErrWorkDirLeftAt           = common.ErrWorkDirLeftAt
	ErrXcodegenGenerate        = common.ErrXcodegenGenerate
)

// Build / toolchain sentinels.
var (
	ErrOutAppRequired     = common.ErrOutAppRequired
	ErrMacOSRequired      = errs.Wrap(errs.ErrRequired, "full .app requires macOS with Xcode; use --go-only")
	ErrXcodebuildNotFound = common.ErrXcodebuildNotFound
	ErrXcodeGenNotFound   = common.ErrXcodeGenNotFound
	ErrDebugAppMissing    = errs.Wrap(errs.ErrMissing, "xcodebuild succeeded but no Debug .app under DerivedData")
	ErrUnsupportedGOARCH  = errs.Wrap(errs.ErrUnsupported, "unsupported host GOARCH for darwin")
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
	// SkipGenerate leaves an existing scaffold in WorkDir in place.
	SkipGenerate bool
	Stdout       io.Writer
	Stderr       io.Writer
}

// BuildResult is the outcome of Build.
type BuildResult struct {
	AppPath    string
	WorkDir    string
	HelperPath string
}

// Build compiles the Go server and, unless GoOnly, builds the scaffold
// with xcodegen and xcodebuild. The .app executable is that shell. The
// Go server is Contents/MacOS/eletrocromo-server. The caller signs the Mach-Os.
func Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	_, stderr := common.Writers(ctx, opts.Stdout, opts.Stderr)

	cfg, err := opts.Config.withDefaults(ctx)
	if err != nil {
		return nil, err
	}
	baseDir, err := common.WorkingDir(opts.BaseDir)
	if err != nil {
		return nil, err
	}
	goMain, err := ResolveGoMain(cfg.GoMain, baseDir)
	if err != nil {
		return nil, err
	}

	vi, name, code := common.StampPackagingVersion(ctx, goMain, opts.Config.VersionName, opts.Config.VersionCode)
	cfg.VersionName = name
	cfg.VersionCode = code
	slog.Info("macos version", "version", cfg.VersionName, "code", cfg.VersionCode)

	workDir, ephemeral, err := common.ResolveWorkDir(opts.WorkDir, "eletrocromo-macos-*")
	if err != nil {
		return nil, err
	}
	var buildErr error
	defer func() {
		common.CleanupWorkDir(ephemeral, buildErr, opts.KeepWorkDir, opts.GoOnly, workDir, stderr, "macos")
	}()

	genCfg := cfg
	genCfg.GoMain = goMain

	if !opts.SkipGenerate {
		slog.Info("macos host", "dir", workDir)
		if err := Create(ctx, Options{OutDir: workDir, Force: true, Config: genCfg}); err != nil {
			buildErr = fmt.Errorf("generate host: %w", err)
			return nil, buildErr
		}
	}

	iconRoot, err := common.IconRoot(baseDir, cfg.Icon, opts.IconRoot, workDir, ".eletrocromo-icons")
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	slog.Info("macos icon", "dir", iconRoot)
	if err := applyMacIcons(iconRoot, filepath.Join(workDir, "Assets.xcassets")); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := icons.ApplyMacOSICNS(iconRoot, filepath.Join(workDir, "Resources", "AppIcon.icns")); err != nil {
		buildErr = err
		return nil, buildErr
	}

	if !opts.GoOnly && runtime.GOOS != "darwin" {
		buildErr = ErrMacOSRequired
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

	xArch, err := darwinXArch(arch)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	slog.Info("macos xcode", "arch", xArch)
	built, err := assembleMac(ctx, workDir, cfg.ProductName(), xArch)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := installHelper(built, helperDest); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := common.ReplaceDir(built, outApp); err != nil {
		buildErr = fmt.Errorf("copy app: %w", err)
		return nil, buildErr
	}
	result.AppPath = outApp
	slog.Info("macos app", "path", outApp)
	return result, nil
}

func installHelper(appPath, helperPath string) error {
	dest := filepath.Join(appPath, "Contents", "MacOS", HelperName)
	if err := common.CopyFile(helperPath, dest, 0o755); err != nil {
		return fmt.Errorf("%w: %w", ErrBundleServer, err)
	}
	return os.Chmod(dest, 0o755)
}

func assembleMac(ctx context.Context, workDir, product, xArch string) (string, error) {
	if err := common.XcodeReady(ctx); err != nil {
		return "", err
	}
	if err := gocmd.Tool(ctx, "xcodegen", workDir, nil, "generate"); err != nil {
		return "", fmt.Errorf("%w: %w", ErrXcodegenGenerate, err)
	}
	derived := filepath.Join(workDir, "build", "DerivedData")
	proj := filepath.Join(workDir, product+".xcodeproj")
	if err := gocmd.Tool(ctx, "xcodebuild", workDir, nil,
		"-project", proj,
		"-scheme", product,
		"-configuration", "Debug",
		"-sdk", "macosx",
		"-destination", "generic/platform=macOS",
		"-derivedDataPath", derived,
		"-skipPackagePluginValidation",
		"CODE_SIGN_IDENTITY=-",
		"CODE_SIGNING_REQUIRED=NO",
		"CODE_SIGNING_ALLOWED=YES",
		"AD_HOC_CODE_SIGNING_ALLOWED=YES",
		"ENABLE_DEBUG_DYLIB=NO",
		"ARCHS="+xArch,
		"ONLY_ACTIVE_ARCH=YES",
		"build",
	); err != nil {
		return "", fmt.Errorf("xcodebuild: %w\n(%w %s)", err, ErrWorkDirLeftAt, workDir)
	}
	app := filepath.Join(derived, "Build", "Products", "Debug", product+".app")
	if st, err := os.Stat(app); err != nil || !st.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrDebugAppMissing, app)
	}
	return app, nil
}

func darwinXArch(goarch string) (string, error) {
	switch goarch {
	case "arm64":
		return "arm64", nil
	case "amd64":
		return "x86_64", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedGOARCH, goarch)
	}
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
		return fmt.Errorf("%w/%s: %w", ErrGoBuildDarwin, goarch, err)
	}
	return os.Chmod(dest, 0o755)
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
		return fmt.Errorf("%w: %w", ErrMacosAppIcon, err)
	}
	if img.Bounds().Dx() < 1 || img.Bounds().Dy() < 1 {
		return ErrMacosAppIconEmptyMaster
	}
	iconDir := filepath.Join(assetsDir, "AppIcon.appiconset")
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return err
	}
	for _, slot := range macIconSlots {
		if err := icons.WritePNG(filepath.Join(iconDir, slot.name), convert.Resize(img, slot.px)); err != nil {
			return err
		}
	}
	return nil
}
