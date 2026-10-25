package host

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/build/gen/apk"
	"github.com/lewtec/lewkit/x/build/gen/ios"
	"github.com/lewtec/lewkit/x/build/gen/linux"
	"github.com/lewtec/lewkit/x/build/gen/mac"
	"github.com/lewtec/lewkit/x/build/gen/win"
	"github.com/lewtec/lewkit/x/build/sign"
)

// AndroidSDK returns an Android SDK root that is already installed.
// The cgo NDK lookup uses it. A missing SDK is an error.
// This does not install one. release run finds adb on PATH.
func AndroidSDK(ctx context.Context) (string, error) {
	return apk.AndroidSDK(ctx)
}

// Host is one packaged app build. Spec is merged with eletrocromo.json.
type Host struct {
	Spec
	Out    string
	Work   string
	GOARCH string
	SDK    string
	GoOnly bool
	CGO    bool
	// IconRoot is a generated icon tree. Empty means Workflow's icons
	// step, or the host build itself, writes one.
	IconRoot string
	// prepared means the scaffold step already wrote the host tree into Work.
	prepared bool
	// Sign is the publisher key. Nil signs the APK with the debug identity
	// cached for this user and ad-hoc signs the macOS and iOS Mach-O.
	// A key signs the APK, the Windows exe, and the Apple Mach-O with that identity.
	Sign *sign.Identity
}

// Android builds an APK from an eletrocromo.json config.
// GoOnly stops after the Android Go library, before the APK is packed.
// Packing does not use the Android SDK. A nil Sign uses the debug identity.
func (host Host) Android(ctx context.Context) (string, error) {
	cfg, base, err := host.Load(ctx)
	if err != nil {
		return "", err
	}
	abi, err := build.AndroidABI(host.GOARCH)
	if err != nil {
		return "", err
	}
	cfg.ABIs = []string{abi}
	result, err := apk.Build(ctx, apk.BuildOptions{
		Config:       cfg,
		BaseDir:      base,
		WorkDir:      host.Work,
		KeepWorkDir:  host.Work != "",
		OutAPK:       host.Out,
		GoOnly:       host.GoOnly,
		CGO:          host.CGO,
		IconRoot:     host.IconRoot,
		SkipGenerate: host.prepared,
	})
	if err != nil {
		return "", err
	}
	if host.GoOnly {
		return result.WorkDir, nil
	}
	signer := host.Sign
	if signer == nil {
		var err error
		signer, err = sign.DebugIdentity(ctx)
		if err != nil {
			return "", err
		}
	}
	if err := signer.SignAPKFile(result.APKPath); err != nil {
		return "", err
	}
	return result.APKPath, nil
}

// Mac builds a .app, then ad-hoc signs that bundle. The Go program is
// the bundle executable on every host and opens app.GUI and app.Web.
// Host.Sign replaces the ad-hoc signature with the publisher key.
func (host Host) Mac(ctx context.Context) (string, error) {
	cfg, base, err := host.Load(ctx)
	if err != nil {
		return "", err
	}
	result, err := mac.Build(ctx, mac.BuildOptions{
		Config:       macConfig(cfg),
		BaseDir:      base,
		WorkDir:      host.Work,
		KeepWorkDir:  host.Work != "",
		OutApp:       host.Out,
		GoOnly:       host.GoOnly,
		GOARCH:       host.GOARCH,
		IconRoot:     host.IconRoot,
		SkipGenerate: host.prepared,
	})
	if err != nil {
		return "", err
	}
	if host.GoOnly {
		return result.WorkDir, nil
	}
	if err := sign.SignTree(host.Sign, result.AppPath, cfg.PackageID); err != nil {
		return "", err
	}
	return result.AppPath, nil
}

// Linux builds one AppImage file. The file is the executable.
// Its trailer holds the app id, icon.png, and a desktop entry.
func (host Host) Linux(ctx context.Context) (string, error) {
	cfg, base, err := host.Load(ctx)
	if err != nil {
		return "", err
	}
	result, err := linux.Build(ctx, linux.BuildOptions{
		Config:      linuxConfig(cfg),
		BaseDir:     base,
		WorkDir:     host.Work,
		KeepWorkDir: host.Work != "",
		OutApp:      host.Out,
		GoOnly:      host.GoOnly,
		GOARCH:      host.GOARCH,
		IconRoot:    host.IconRoot,
	})
	if err != nil {
		return "", err
	}
	return result.AppPath, nil
}

// Windows builds a GUI .exe. The binary is windowsgui, with the app icon,
// a PerMonitorV2 manifest, and version info.
func (host Host) Windows(ctx context.Context) (string, error) {
	cfg, base, err := host.Load(ctx)
	if err != nil {
		return "", err
	}
	result, err := win.Build(ctx, win.BuildOptions{
		Config:      winConfig(cfg),
		BaseDir:     base,
		WorkDir:     host.Work,
		KeepWorkDir: host.Work != "",
		OutExe:      host.Out,
		GoOnly:      host.GoOnly,
		GOARCH:      host.GOARCH,
		IconRoot:    host.IconRoot,
	})
	if err != nil {
		return "", err
	}
	if host.GoOnly {
		return result.WorkDir, nil
	}
	if host.Sign != nil {
		if err := host.Sign.SignPEFile(ctx, result.ExePath, cfg.AppName); err != nil {
			return "", err
		}
	}
	return result.ExePath, nil
}

// IOS builds a .app around a cgo-free GOOS=ios program on every host.
// An Out path that ends in .ipa signs that bundle, then writes
// Payload/<product>.app into the archive. iphonesimulator stamps the
// simulator platform, including arm64. A nil Host.Sign ad-hoc signs,
// the same as Mac. The simulator refuses an unsigned device binary.
func (host Host) IOS(ctx context.Context) (string, error) {
	cfg, base, err := host.Load(ctx)
	if err != nil {
		return "", err
	}
	outApp := host.Out
	var scratch string
	if strings.HasSuffix(host.Out, ".ipa") && !host.GoOnly {
		scratch, err = os.MkdirTemp("", "lewkit-ipa-")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(scratch)
		outApp = filepath.Join(scratch, iosConfig(cfg).ProductName()+".app")
	}
	result, err := ios.Build(ctx, ios.BuildOptions{
		Config:       iosConfig(cfg),
		BaseDir:      base,
		WorkDir:      host.Work,
		KeepWorkDir:  host.Work != "",
		OutApp:       outApp,
		GoOnly:       host.GoOnly,
		SDK:          host.SDK,
		GOARCH:       host.GOARCH,
		IconRoot:     host.IconRoot,
		SkipGenerate: host.prepared,
	})
	if err != nil {
		return "", err
	}
	if host.GoOnly {
		return result.WorkDir, nil
	}
	if err := sign.SignTree(host.Sign, result.AppPath, cfg.PackageID); err != nil {
		return "", err
	}
	if scratch == "" {
		return result.AppPath, nil
	}
	if err := ios.Pack(ctx, result.AppPath, host.Out); err != nil {
		return "", err
	}
	return host.Out, nil
}

// ArtifactName is the file inside a dist directory for one host package.
// iOS is product_ios_goarch.app. iphoneos is product_ios_goarch.ipa.
// An empty SDK is the simulator name.
func ArtifactName(product, packageID, goos, goarch, sdk string) (string, error) {
	if build.GOOS(goos) != build.GOOSIOS {
		return build.AppFile(product, packageID, goos, goarch), nil
	}
	device, err := iosDevice(sdk)
	if err != nil {
		return "", err
	}
	return build.IOSFile(product, goarch, device), nil
}

func iosDevice(sdk string) (bool, error) {
	parsed, err := ios.ParseSDK(sdk)
	if err != nil {
		return false, err
	}
	return parsed == ios.SDKDevice, nil
}

func macConfig(cfg apk.Config) mac.Config {
	return mac.Config{
		PackageID:    cfg.PackageID,
		AppName:      cfg.AppName,
		VersionName:  cfg.VersionName,
		VersionCode:  cfg.VersionCode,
		GoMain:       cfg.GoMain,
		Icon:         cfg.Icon,
		Capabilities: cfg.Capabilities,
	}
}

func linuxConfig(cfg apk.Config) linux.Config {
	return linux.Config{
		PackageID:    cfg.PackageID,
		AppName:      cfg.AppName,
		VersionName:  cfg.VersionName,
		VersionCode:  cfg.VersionCode,
		GoMain:       cfg.GoMain,
		Icon:         cfg.Icon,
		Capabilities: cfg.Capabilities,
	}
}

func winConfig(cfg apk.Config) win.Config {
	return win.Config{
		PackageID:    cfg.PackageID,
		AppName:      cfg.AppName,
		VersionName:  cfg.VersionName,
		VersionCode:  cfg.VersionCode,
		GoMain:       cfg.GoMain,
		Icon:         cfg.Icon,
		Capabilities: cfg.Capabilities,
	}
}

func iosConfig(cfg apk.Config) ios.Config {
	return ios.Config{
		PackageID:    cfg.PackageID,
		AppName:      cfg.AppName,
		VersionName:  cfg.VersionName,
		VersionCode:  cfg.VersionCode,
		GoMain:       cfg.GoMain,
		Icon:         cfg.Icon,
		Capabilities: cfg.Capabilities,
	}
}
