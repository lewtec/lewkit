package ios

import (
	"context"
	"encoding/json"
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
	ErrDarwinRequired     = errors.New("ios archive and .app require macOS with Xcode; use --go-only on a Mac")
	ErrXcodebuildNotFound = errors.New("xcodebuild not found; install Xcode")
	ErrXcodeGenNotFound   = errors.New("xcodegen not found; install xcodegen (mise/brew)")
	ErrDebugAppMissing    = errors.New("xcodebuild succeeded but no Debug-iphonesimulator .app under DerivedData")
	ErrUnknownSDK         = errors.New("sdk must be iphonesimulator or iphoneos")
)

// SDKSimulator / SDKDevice are xcrun SDK names.
const (
	SDKSimulator = "iphonesimulator"
	SDKDevice    = "iphoneos"
)

// BuildOptions drives an iOS .app build from an eletrocromo app.
type BuildOptions struct {
	Config      Config
	BaseDir     string
	WorkDir     string
	KeepWorkDir bool
	OutApp      string
	GoOnly      bool
	SDK         string
	GOARCH      string
	IconRoot    string
	Stdout      io.Writer
	Stderr      io.Writer
}

// BuildResult is the outcome of Build.
type BuildResult struct {
	AppPath     string
	WorkDir     string
	ArchivePath string
}

// Build scaffolds the iOS host, builds a GOOS=ios c-archive, and (unless
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

	sdk, err := normalizeSDK(opts.SDK)
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
		return nil, ErrDarwinRequired
	}

	workDir, ephemeral, err := common.ResolveWorkDir(opts.WorkDir, "eletrocromo-ios-*")
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

	if _, err := fmt.Fprintf(stdout, "eletrocromo: generating iOS host in %s\n", workDir); err != nil {
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
	assetsDir := filepath.Join(workDir, "Assets.xcassets")
	if _, err := fmt.Fprintf(stdout, "eletrocromo: applying iOS icon from %s\n", iconRoot); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := applyIOSIcons(iconRoot, assetsDir); err != nil {
		buildErr = err
		return nil, buildErr
	}

	archiveDest := filepath.Join(workDir, "lib", ArchiveName+".a")
	result := &BuildResult{WorkDir: workDir}

	if runtime.GOOS != "darwin" {
		if _, err := fmt.Fprintf(stdout, "eletrocromo: --go-only: skipped ios c-archive (need macOS + Xcode)\n"); err != nil {
			return result, err
		}
		return result, nil
	}

	if _, err := fmt.Fprintf(stdout, "eletrocromo: building Go c-archive (GOOS=ios SDK=%s)\n", sdk); err != nil {
		buildErr = err
		return nil, buildErr
	}
	goarch, xArch := hostIOSArch(sdk)
	if opts.GOARCH != "" {
		var archErr error
		goarch, xArch, archErr = iosArch(opts.GOARCH)
		if archErr != nil {
			buildErr = archErr
			return nil, buildErr
		}
	}
	if err := buildArchive(ctx, archiveDest, goMain, workDir, sdk, vi, cfg.PackageID, goarch); err != nil {
		buildErr = err
		return nil, buildErr
	}
	result.ArchivePath = archiveDest

	if opts.GoOnly {
		_, err := fmt.Fprintf(stdout, "eletrocromo: --go-only: skipped xcodebuild (archive %s)\n", archiveDest)
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
	built, err := assembleDebug(workDir, cfg.ProductName(), sdk, xArch, stdout, stderr)
	if err != nil {
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

func normalizeSDK(sdk string) (string, error) {
	s := strings.TrimSpace(sdk)
	if s == "" {
		return SDKSimulator, nil
	}
	switch s {
	case SDKSimulator, SDKDevice, "simulator":
		if s == "simulator" {
			return SDKSimulator, nil
		}
		return s, nil
	case "device":
		return SDKDevice, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownSDK, s)
	}
}

// SplashPointSize is the on-screen logo size (points) for the launch
// storyboard and the in-app splash. Asset catalog 1x/2x/3x files match this
// so UILaunchScreen does not treat a 1024px PNG as a 1024pt image.
const SplashPointSize = 120

func applyIOSIcons(iconRoot, assetsDir string) error {
	src := filepath.Join(iconRoot, "source", "master.png")
	img, err := icons.DecodeImage(src)
	if err != nil {
		return fmt.Errorf("ios app icon: %w", err)
	}
	square := icons.KnockoutBackground(icons.PadCenter(img))
	store := icons.FlattenOpaque(icons.Resize(square, 1024))
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
		if err := icons.WritePNG(filepath.Join(logoDir, scale.name), icons.Resize(square, scale.px)); err != nil {
			return err
		}
	}
	return nil
}

func buildArchive(ctx context.Context, dest, goMainDir, workDir, sdk string, stamp version.Info, appID, goarch string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	bridgePath := filepath.Join(workDir, BridgeGoName)
	if err := os.WriteFile(bridgePath, []byte(iosBridgeSource), 0o644); err != nil {
		return err
	}
	virtual := filepath.Join(goMainDir, BridgeGoName)
	overlayDoc := map[string]map[string]string{
		"Replace": {virtual: bridgePath},
	}
	overlayRaw, err := json.Marshal(overlayDoc)
	if err != nil {
		return err
	}
	overlayFile := filepath.Join(workDir, "overlay.json")
	if err := os.WriteFile(overlayFile, overlayRaw, 0o644); err != nil {
		return err
	}

	wrap, err := writeClangwrap(workDir)
	if err != nil {
		return err
	}

	err = gocmd.Command{
		Dir: goMainDir,
		Env: append(os.Environ(),
			"CGO_ENABLED=1",
			"GOOS=ios",
			"GOARCH="+goarch,
			"CC="+wrap,
			"ELETROCROMO_IOS_SDK="+sdk,
			"CGO_LDFLAGS=-framework CoreFoundation",
		),
		Args: []string{
			"-buildmode=c-archive",
			"-trimpath",
			"-overlay", overlayFile,
			"-ldflags", stamp.WithAppID(appID),
			"-o", dest,
			".",
		},
	}.Run(ctx)
	if err != nil {
		return fmt.Errorf("go build ios/%s (%s): %w", goarch, sdk, err)
	}
	hdr := strings.TrimSuffix(dest, ".a") + ".h"
	if _, err := os.Stat(hdr); err != nil {
		return fmt.Errorf("c-archive header: %w", err)
	}
	return nil
}

func writeClangwrap(workDir string) (string, error) {
	raw, err := templateFS.ReadFile("clangwrap.sh")
	if err != nil {
		return "", err
	}
	dest := filepath.Join(workDir, "clangwrap.sh")
	if err := os.WriteFile(dest, raw, 0o755); err != nil {
		return "", err
	}
	return dest, nil
}

func assembleDebug(workDir, product, sdk, xArch string, stdout, stderr io.Writer) (string, error) {
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

	dest, destName := xcodeDestination(sdk)
	derived := filepath.Join(workDir, "build", "DerivedData")
	proj := filepath.Join(workDir, product+".xcodeproj")
	cmd := exec.Command("xcodebuild",
		"-project", proj,
		"-scheme", product,
		"-configuration", "Debug",
		"-sdk", sdk,
		"-destination", dest,
		"-derivedDataPath", derived,
		"-skipPackagePluginValidation",
		"CODE_SIGN_IDENTITY=-",
		"CODE_SIGNING_REQUIRED=NO",
		"CODE_SIGNING_ALLOWED=YES",
		"AD_HOC_CODE_SIGNING_ALLOWED=YES",
		"ENABLE_DEBUG_DYLIB=NO",
		"ARCHS="+xArch,
		"ONLY_ACTIVE_ARCH=YES",
		"EXCLUDED_ARCHS="+excludedArch(xArch),
		"build",
	)
	cmd.Dir = workDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("xcodebuild: %w\n(work dir left at %s)", err, workDir)
	}

	app := filepath.Join(derived, "Build", "Products", destName, product+".app")
	if st, err := os.Stat(app); err != nil || !st.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrDebugAppMissing, app)
	}
	return app, nil
}

func iosArch(goarch string) (string, string, error) {
	switch goarch {
	case "arm64":
		return "arm64", "arm64", nil
	case "amd64":
		return "amd64", "x86_64", nil
	default:
		return "", "", fmt.Errorf("unsupported GOARCH %q", goarch)
	}
}

func hostIOSArch(sdk string) (goarch, xArch string) {
	arch, xarch, err := iosArch(runtime.GOARCH)
	if err != nil || sdk == "" {
		return "arm64", "arm64"
	}
	return arch, xarch
}

func excludedArch(xArch string) string {
	if xArch == "arm64" {
		return "x86_64"
	}
	return "arm64"
}

func xcodeDestination(sdk string) (destination, productsDir string) {
	if sdk == SDKDevice {
		return "generic/platform=iOS", "Debug-iphoneos"
	}
	return "generic/platform=iOS Simulator", "Debug-iphonesimulator"
}
