package build

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/build/gen/apk"
	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/gen/ios"
	"github.com/lewtec/lewkit/x/build/gen/mac"
)

// Host is one packaged app build.
type Host struct {
	Config string
	Out    string
	Work   string
	GOARCH string
	SDK    string
	GoOnly bool
}

// Android builds an APK from an eletrocromo.json config.
// GoOnly stops after the Android Go library, before Gradle.
func (host Host) Android(ctx context.Context) (string, error) {
	cfg, base, err := apk.LoadConfig(host.Config)
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
	})
	if err != nil {
		return "", err
	}
	if host.GoOnly {
		return result.WorkDir, nil
	}
	return result.APKPath, nil
}

// Mac builds an unsigned Debug .app from an eletrocromo.json config.
func (host Host) Mac(ctx context.Context) (string, error) {
	cfg, base, err := loadHost(host.Config)
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
	cfg, base, err := loadHost(host.Config)
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

type hostFile struct {
	PackageID    string          `json:"package_id"`
	AppName      string          `json:"app_name"`
	VersionName  string          `json:"version_name"`
	VersionCode  int             `json:"version_code"`
	GoMain       string          `json:"go_main"`
	Icon         string          `json:"icon"`
	Capabilities json.RawMessage `json:"capabilities"`
}

func loadHost(path string) (common.HostConfig, string, error) {
	cfg, base, err := apk.LoadConfig(path)
	if err != nil {
		return common.HostConfig{}, "", err
	}
	raw, err := os.ReadFile(configFile(path))
	if err != nil {
		return common.HostConfig{}, "", err
	}
	var doc hostFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		return common.HostConfig{}, "", err
	}
	caps, err := common.ParseCapabilities(doc.Capabilities)
	if err != nil {
		return common.HostConfig{}, "", err
	}
	host := common.HostConfig{
		PackageID:    cfg.PackageID,
		AppName:      cfg.AppName,
		VersionName:  cfg.VersionName,
		VersionCode:  cfg.VersionCode,
		GoMain:       cfg.GoMain,
		Icon:         cfg.Icon,
		Capabilities: caps,
	}
	return host, base, nil
}

func configFile(path string) string {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return path
	}
	return filepath.Join(path, apk.ConfigFileName)
}

func macConfig(host common.HostConfig) mac.Config {
	return mac.Config{
		PackageID:    host.PackageID,
		AppName:      host.AppName,
		VersionName:  host.VersionName,
		VersionCode:  host.VersionCode,
		GoMain:       host.GoMain,
		Icon:         host.Icon,
		Capabilities: host.Capabilities,
	}
}

func iosConfig(host common.HostConfig) ios.Config {
	return ios.Config{
		PackageID:    host.PackageID,
		AppName:      host.AppName,
		VersionName:  host.VersionName,
		VersionCode:  host.VersionCode,
		GoMain:       host.GoMain,
		Icon:         host.Icon,
		Capabilities: host.Capabilities,
	}
}
