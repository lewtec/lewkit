// Package ios generates an iOS .app.
// The Go program is the bundle executable on every host.
// Create writes the scaffold. Build does not compile that scaffold.
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

func (c Config) hostConfig() common.HostConfig { return common.HostConfig(c) }

func configFromHost(id common.HostConfig) Config { return Config(id) }

func (c Config) withDefaults(ctx context.Context) (Config, error) {
	return common.WithDefaults(ctx, c, Config.hostConfig, configFromHost)
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
