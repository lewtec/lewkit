// Package mac packages a macOS .app from the XcodeGen scaffold.
// The launched executable replaces itself with the Go program beside it.
// That program opens app.GUI and app.Web. Create writes the scaffold.
// Build compiles it with xcodebuild unless GoOnly is set.
package mac

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

// HelperName is the Go program copied into Contents/MacOS.
// The scaffold executable replaces itself with this program.
// app.GUI and app.Web in that process open the windows.
const HelperName = "eletrocromo-server"

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
	return common.EncodeHostJSON(cfg.hostConfig(), "eletrocromo-macos")
}

// ResolveGoMain returns an absolute directory containing the Go main package.
func ResolveGoMain(goMain, baseDir string) (string, error) {
	return common.ResolveGoMain(goMain, baseDir)
}

// DefaultOutApp is dist/<app_name>.app under cwd.
func DefaultOutApp(appName, cwd string) string {
	return common.DefaultOutApp(appName, cwd)
}
