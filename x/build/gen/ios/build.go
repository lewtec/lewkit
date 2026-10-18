package ios

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
	ErrApplyIOSIconsIosAppIcon = errors.New("ios app icon")
	ErrBuildArchiveGoBuildIos  = errors.New("go build ios")
	ErrBundleExecutable        = errors.New("bundle executable")
	ErrUnsupportedGoarch       = errs.Wrap(errs.ErrUnsupported, "unsupported GOARCH")
)

// Build / toolchain sentinels.
var (
	ErrOutAppRequired = common.ErrOutAppRequired
	ErrIOSPlatform    = errs.Wrap(errs.ErrInvalid, "ios executable platform")
)

// SDK is an xcrun SDK name. An empty text is the simulator.
//
//lewkit:enum empty=iphonesimulator
type SDK string

const (
	// SDKSimulator is the iOS Simulator SDK.
	SDKSimulator SDK = "iphonesimulator" // simulator
	// SDKDevice is the iOS device SDK.
	SDKDevice SDK = "iphoneos" // device
)

// BuildOptions drives an iOS .app build from an eletrocromo app.
type BuildOptions struct {
	Config       Config
	BaseDir      string
	WorkDir      string
	KeepWorkDir  bool
	OutApp       string
	GoOnly       bool
	SDK          string
	GOARCH       string
	IconRoot     string
	SkipGenerate bool
	Stdout       io.Writer
	Stderr       io.Writer
}

// BuildResult is the outcome of Build.
type BuildResult struct {
	AppPath     string
	WorkDir     string
	ArchivePath string
}

// Build scaffolds the iOS host and packages a .app.
// Every host assembles the bundle around a cgo-free GOOS=ios executable
// and stamps LC_BUILD_VERSION. The caller signs the Mach-O.
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

	if _, err := normalizeSDK(opts.SDK); err != nil {
		return nil, err
	}

	vi, name, code := common.StampPackagingVersion(ctx, goMain, opts.Config.VersionName, opts.Config.VersionCode)
	cfg.VersionName = name
	cfg.VersionCode = code
	slog.Info("ios version", "version", cfg.VersionName, "code", cfg.VersionCode)

	workDir, ephemeral, err := common.ResolveWorkDir(opts.WorkDir, "eletrocromo-ios-*")
	if err != nil {
		return nil, err
	}
	var buildErr error
	defer func() {
		common.CleanupWorkDir(ephemeral, buildErr, opts.KeepWorkDir, opts.GoOnly, workDir, stderr, "ios")
	}()

	genCfg := cfg
	genCfg.GoMain = goMain

	if !opts.SkipGenerate {
		slog.Info("ios host", "dir", workDir)
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
	assetsDir := filepath.Join(workDir, "Assets.xcassets")
	slog.Info("ios icon", "dir", iconRoot)
	if err := applyIOSIcons(iconRoot, assetsDir); err != nil {
		buildErr = err
		return nil, buildErr
	}

	built, err := assembleGoIOS(ctx, opts, cfg, goMain, workDir, iconRoot, vi)
	buildErr = err
	return built, err
}

func normalizeSDK(sdk string) (string, error) {
	parsed, err := ParseSDK(sdk)
	if err != nil {
		return "", err
	}
	return string(parsed), nil
}

// SplashPointSize is the on-screen logo size (points) for the launch
// storyboard and the in-app splash. Asset catalog 1x/2x/3x files match this
// so UILaunchScreen does not treat a 1024px PNG as a 1024pt image.
const SplashPointSize = 120

func applyIOSIcons(iconRoot, assetsDir string) error {
	src := filepath.Join(iconRoot, "source", "master.png")
	img, err := icons.DecodeImage(src)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrApplyIOSIconsIosAppIcon, err)
	}
	square := icons.KnockoutBackground(convert.Pad(img))
	store := icons.FlattenOpaque(convert.Resize(square, 1024))
	iconDir := filepath.Join(assetsDir, "AppIcon.appiconset")
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		return err
	}
	if err := icons.WritePNG(filepath.Join(iconDir, "AppIcon.png"), store); err != nil {
		return err
	}
	logoDir := filepath.Join(assetsDir, "SplashLogo.imageset")
	if err := os.MkdirAll(logoDir, 0o755); err != nil {
		return err
	}
	for _, scale := range []struct {
		name string
		px   int
	}{
		{"SplashLogo.png", SplashPointSize},
		{"SplashLogo@2x.png", SplashPointSize * 2},
		{"SplashLogo@3x.png", SplashPointSize * 3},
	} {
		if err := icons.WritePNG(filepath.Join(logoDir, scale.name), convert.Resize(square, scale.px)); err != nil {
			return err
		}
	}
	return nil
}

func assembleGoIOS(ctx context.Context, opts BuildOptions, cfg Config, goMain, workDir, iconRoot string, stamp version.Info) (*BuildResult, error) {
	goarch := "arm64"
	if opts.GOARCH != "" {
		parsed, _, err := iosArch(opts.GOARCH)
		if err != nil {
			return nil, err
		}
		goarch = parsed
	} else if parsed, _, err := iosArch(runtime.GOARCH); err == nil {
		goarch = parsed
	}
	exe := filepath.Join(workDir, "bin", cfg.ProductName())
	slog.Info("ios go", "arch", goarch)
	if err := buildIOSExe(ctx, exe, goMain, goarch, stamp, cfg.PackageID); err != nil {
		return nil, err
	}
	if err := stampPlatform(exe, goarch); err != nil {
		return nil, err
	}
	result := &BuildResult{WorkDir: workDir, ArchivePath: exe}
	if opts.GoOnly {
		slog.Info("ios go-only", "exe", exe)
		return result, nil
	}
	outApp := strings.TrimSpace(opts.OutApp)
	if outApp == "" {
		return nil, ErrOutAppRequired
	}
	outApp, err := filepath.Abs(outApp)
	if err != nil {
		return nil, err
	}
	if err := writeIOSBundle(cfg, iconRoot, exe, outApp, goarch); err != nil {
		return nil, err
	}
	result.AppPath = outApp
	slog.Info("ios app", "path", outApp)
	return result, nil
}

// buildIOSExe links a cgo-free GOOS=ios executable with the embedded
// toolchain. Host go refuses ios/arm64 unless cgo is enabled.
func buildIOSExe(ctx context.Context, dest, goMainDir, goarch string, stamp version.Info, appID string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	err := gocmd.Command{
		Compiler: gocmd.CompilerLewkit,
		Dir:      goMainDir,
		Env: append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS=ios",
			"GOARCH="+goarch,
		),
		Args: []string{"-trimpath", "-ldflags", stamp.WithAppID(appID), "-o", dest, "."},
	}.Run(ctx)
	if err != nil {
		return fmt.Errorf("%w/%s: %w", ErrBuildArchiveGoBuildIos, goarch, err)
	}
	return os.Chmod(dest, 0o755)
}

func writeIOSBundle(cfg Config, iconRoot, exe, outApp, goarch string) error {
	product := cfg.ProductName()
	stage, err := os.MkdirTemp("", "eletrocromo-ios-app-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	app := filepath.Join(stage, product+".app")
	if err := os.MkdirAll(app, 0o755); err != nil {
		return err
	}
	if err := common.CopyFile(exe, filepath.Join(app, product), 0o755); err != nil {
		return fmt.Errorf("%w: %w", ErrBundleExecutable, err)
	}
	if err := os.WriteFile(filepath.Join(app, "Info.plist"), []byte(iosPlist(cfg, goarch)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(app, "PkgInfo"), []byte("APPL????"), 0o644); err != nil {
		return err
	}
	if err := writeBundleIcons(iconRoot, app); err != nil {
		return err
	}
	return common.ReplaceDir(app, outApp)
}

func iosPlist(cfg Config, goarch string) string {
	platform := "iPhoneOS"
	if goarch == "amd64" {
		platform = "iPhoneSimulator"
	}
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")
	plistString(&b, "CFBundleDevelopmentRegion", "en")
	plistString(&b, "CFBundleDisplayName", cfg.AppName)
	plistString(&b, "CFBundleExecutable", cfg.ProductName())
	plistString(&b, "CFBundleIdentifier", cfg.PackageID)
	plistString(&b, "CFBundleInfoDictionaryVersion", "6.0")
	plistString(&b, "CFBundleName", cfg.AppName)
	plistString(&b, "CFBundlePackageType", "APPL")
	plistString(&b, "CFBundleShortVersionString", cfg.VersionName)
	plistString(&b, "CFBundleVersion", fmt.Sprintf("%d", cfg.VersionCode))
	plistString(&b, "MinimumOSVersion", "13.0")
	b.WriteString("\t<key>LSRequiresIPhoneOS</key>\n\t<true/>\n")
	b.WriteString("\t<key>CFBundleSupportedPlatforms</key>\n\t<array>\n\t\t<string>")
	b.WriteString(platform)
	b.WriteString("</string>\n\t</array>\n")
	b.WriteString("\t<key>UIDeviceFamily</key>\n\t<array>\n\t\t<integer>1</integer>\n\t\t<integer>2</integer>\n\t</array>\n")
	b.WriteString("\t<key>UILaunchScreen</key>\n\t<dict/>\n")
	b.WriteString("\t<key>UISupportedInterfaceOrientations</key>\n\t<array>\n")
	b.WriteString("\t\t<string>UIInterfaceOrientationPortrait</string>\n")
	b.WriteString("\t\t<string>UIInterfaceOrientationLandscapeLeft</string>\n")
	b.WriteString("\t\t<string>UIInterfaceOrientationLandscapeRight</string>\n")
	b.WriteString("\t</array>\n")
	b.WriteString("\t<key>CFBundleIcons</key>\n\t<dict>\n\t\t<key>CFBundlePrimaryIcon</key>\n\t\t<dict>\n")
	b.WriteString("\t\t\t<key>CFBundleIconFiles</key>\n\t\t\t<array>\n\t\t\t\t<string>Icon</string>\n\t\t\t</array>\n")
	b.WriteString("\t\t</dict>\n\t</dict>\n")
	if goarch == "arm64" {
		b.WriteString("\t<key>UIRequiredDeviceCapabilities</key>\n\t<array>\n\t\t<string>arm64</string>\n\t</array>\n")
	}
	b.WriteString(cfg.Capabilities.PlistURLTypes(cfg.PackageID))
	b.WriteString(cfg.Capabilities.PlistDocumentTypes())
	b.WriteString("</dict>\n</plist>\n")
	return b.String()
}

func plistString(b *strings.Builder, key, value string) {
	b.WriteString("\t<key>")
	b.WriteString(key)
	b.WriteString("</key>\n\t<string>")
	b.WriteString(common.XMLEscape(value))
	b.WriteString("</string>\n")
}

func writeBundleIcons(iconRoot, appDir string) error {
	src := filepath.Join(iconRoot, "source", "master.png")
	img, err := icons.DecodeImage(src)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrApplyIOSIconsIosAppIcon, err)
	}
	square := icons.KnockoutBackground(convert.Pad(img))
	for _, slot := range []struct {
		name string
		px   int
	}{
		{"Icon@2x.png", 120},
		{"Icon@3x.png", 180},
	} {
		if err := icons.WritePNG(filepath.Join(appDir, slot.name), convert.Resize(square, slot.px)); err != nil {
			return err
		}
	}
	return nil
}

func iosArch(goarch string) (string, string, error) {
	switch goarch {
	case "arm64":
		return "arm64", "arm64", nil
	case "amd64":
		return "amd64", "x86_64", nil
	default:
		return "", "", fmt.Errorf("%w %q", ErrUnsupportedGoarch, goarch)
	}
}
