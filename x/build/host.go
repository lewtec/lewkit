package build

import (
	"context"

	"github.com/lewtec/lewkit/x/build/gen/apk"
	"github.com/lewtec/lewkit/x/build/gen/ios"
	"github.com/lewtec/lewkit/x/build/gen/mac"
)

// AndroidSDK returns the Android SDK root used by the host build and adb.
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
}

// Android builds an APK from an eletrocromo.json config.
// GoOnly stops after the Android Go library, before Gradle.
func (host Host) Android(ctx context.Context) (string, error) {
	cfg, base, err := host.Load()
	if err != nil {
		return "", err
	}
	abi, err := AndroidABI(host.GOARCH)
	if err != nil {
		return "", err
	}
	cfg.ABIs = []string{abi}
	result, err := apk.Build(ctx, apk.BuildOptions{
		Config:      cfg,
		BaseDir:     base,
		WorkDir:     host.Work,
		KeepWorkDir: host.Work != "",
		OutAPK:      host.Out,
		GoOnly:      host.GoOnly,
		CGO:         host.CGO,
	})
	if err != nil {
		return "", err
	}
	if host.GoOnly {
		return result.WorkDir, nil
	}
	return result.APKPath, nil
}

// Mac builds an unsigned .app whose executable is the Go program.
func (host Host) Mac(ctx context.Context) (string, error) {
	cfg, base, err := host.Load()
	if err != nil {
		return "", err
	}
	result, err := mac.Build(ctx, mac.BuildOptions{
		Config:      macConfig(cfg),
		BaseDir:     base,
		WorkDir:     host.Work,
		KeepWorkDir: host.Work != "",
		OutApp:      host.Out,
		GoOnly:      host.GoOnly,
		GOARCH:      host.GOARCH,
	})
	if err != nil {
		return "", err
	}
	if host.GoOnly {
		return result.WorkDir, nil
	}
	return result.AppPath, nil
}

// IOS builds a simulator or device .app from an eletrocromo.json config.
func (host Host) IOS(ctx context.Context) (string, error) {
	cfg, base, err := host.Load()
	if err != nil {
		return "", err
	}
	result, err := ios.Build(ctx, ios.BuildOptions{
		Config:      iosConfig(cfg),
		BaseDir:     base,
		WorkDir:     host.Work,
		KeepWorkDir: host.Work != "",
		OutApp:      host.Out,
		GoOnly:      host.GoOnly,
		SDK:         host.SDK,
		GOARCH:      host.GOARCH,
	})
	if err != nil {
		return "", err
	}
	if host.GoOnly {
		return result.WorkDir, nil
	}
	return result.AppPath, nil
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
