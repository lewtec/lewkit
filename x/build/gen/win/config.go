// Package win packages a CGo-less windows Go binary as a GUI .exe.
// The executable is windowsgui, with the app icon, a PerMonitorV2 manifest,
// and version info. Build does not need a Windows SDK.
package win

import (
	"context"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

// Config is the project identity stamped into the executable.
type Config struct {
	PackageID    string              `json:"package_id"`
	AppName      string              `json:"app_name"`
	VersionName  string              `json:"version_name"`
	VersionCode  int                 `json:"version_code"`
	GoMain       string              `json:"go_main"`
	Icon         string              `json:"icon,omitempty"`
	Capabilities common.Capabilities `json:"capabilities,omitempty"`
}

// ProductName is the filesystem-safe executable stem.
func (c Config) ProductName() string {
	return common.ProductName(c.PackageID, c.AppName)
}

func (c Config) hostConfig() common.HostConfig {
	return common.HostConfig{
		PackageID:    c.PackageID,
		AppName:      c.AppName,
		VersionName:  c.VersionName,
		VersionCode:  c.VersionCode,
		GoMain:       c.GoMain,
		Icon:         c.Icon,
		Capabilities: c.Capabilities,
	}
}

func configFromHost(id common.HostConfig) Config {
	return Config{
		PackageID:    id.PackageID,
		AppName:      id.AppName,
		VersionName:  id.VersionName,
		VersionCode:  id.VersionCode,
		GoMain:       id.GoMain,
		Icon:         id.Icon,
		Capabilities: id.Capabilities,
	}
}

func (c Config) withDefaults(ctx context.Context) (Config, error) {
	id, err := common.ApplyHostDefaults(ctx, c.hostConfig())
	if err != nil {
		return Config{}, err
	}
	return configFromHost(id), nil
}

// ResolveGoMain returns an absolute directory containing the Go main package.
func ResolveGoMain(goMain, baseDir string) (string, error) {
	return common.ResolveGoMain(goMain, baseDir)
}
