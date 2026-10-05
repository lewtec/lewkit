// Package ios generates an ephemeral iOS host and packages
// a GOOS=ios c-archive into a Debug .app.
// A GUI model draws on a UIView. A web handler stays in the WKWebView.
//
// iOS cannot exec a helper binary. The Go app is linked in-process
// (EletrocromoStart). A web handler writes ELETROCROMO_READY_FILE.
// A GUI model calls openSurface and does not write that file.
// The importable eletrocromo library stays free of Xcode.
package ios

import (
	"context"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

// Config / path sentinels.
var (
	ErrConfigPathEmpty = common.ErrConfigPathEmpty
	ErrGoMainNotDir    = common.ErrGoMainNotDir
	ErrOutDirRequired  = common.ErrOutDirRequired
	ErrOutPathNotDir   = common.ErrOutPathNotDir
	ErrOutDirNotEmpty  = common.ErrOutDirNotEmpty
)

// ArchiveName is the c-archive stem written under lib/.
const ArchiveName = "libeletrocromo"

// BridgeGoName is overlaid into the app main package during the archive build.
const BridgeGoName = "eletrocromo_ios_bridge.go"

// Config is the project identity written into the generated tree.
type Config struct {
	PackageID    string              `json:"package_id"`
	AppName      string              `json:"app_name"`
	VersionName  string              `json:"version_name"`
	VersionCode  int                 `json:"version_code"`
	GoMain       string              `json:"go_main"`
	Icon         string              `json:"icon,omitempty"`
	Capabilities common.Capabilities `json:"capabilities,omitempty"`
}

// ProductName is a filesystem-safe Xcode PRODUCT_NAME / .app stem.
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

func encodeConfigJSON(cfg Config) ([]byte, error) {
	return common.EncodeHostJSON(cfg.hostConfig(), "eletrocromo-ios")
}

// ResolveGoMain returns an absolute directory containing the Go main package.
func ResolveGoMain(goMain, baseDir string) (string, error) {
	return common.ResolveGoMain(goMain, baseDir)
}

// DefaultOutApp is dist/<app_name>.app under cwd.
func DefaultOutApp(appName, cwd string) string {
	return common.DefaultOutApp(appName, cwd)
}
