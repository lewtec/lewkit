// Package apk generates an Android host project from an embedded template,
// keyed by reverse-domain package ID.
//
// The core eletrocromo library stays free of the Android SDK. This package
// writes a Gradle tree that runs the Go binary. "ui": "surface" launches the
// native window. "ui": "web" keeps the loopback page.
package apk

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/version"
	"github.com/lewtec/lewkit/x/release"
)

//go:embed all:template
var templateFS embed.FS

// Create / out-dir sentinels.
var (
	ErrOutDirRequired = common.ErrOutDirRequired
	ErrOutPathNotDir  = common.ErrOutPathNotDir
	ErrOutDirNotEmpty = common.ErrOutDirNotEmpty
)

// Config is the project identity written into the generated tree and
// eletrocromo.json (re-run / rebuild input).
type Config struct {
	// PackageID is the Android applicationId and Java package (App.ID).
	PackageID string `json:"package_id"`
	// AppName is the launcher label.
	AppName string `json:"app_name"`
	// VersionName is Android versionName. Empty → derived (git describe / -X / devel).
	VersionName string `json:"version_name"`
	// VersionCode is Android versionCode. Empty/0 → semver map or git rev-list count.
	VersionCode int `json:"version_code"`
	// GoMain is a path (often relative to the config file or generated host)
	// to the Go main package directory.
	GoMain string `json:"go_main"`
	// Icon is an optional path to a master PNG/JPEG (relative to config dir).
	// Empty → packaging CLI uses the embedded default mark.
	Icon string `json:"icon,omitempty"`
	// ABIs lists Android ABIs for the fat APK (default DefaultABIs).
	ABIs []string `json:"abis,omitempty"`
	// Capabilities is the closed catalog (url, files) emitted into the host manifest.
	Capabilities common.Capabilities `json:"capabilities,omitempty"`
	// UI is the Android first window. "surface" is the native window.
	// Empty and "web" keep the loopback page.
	UI string `json:"ui,omitempty"`
}

// templateData is passed to text/template for file bodies.
type templateData struct {
	Config
	// RootProjectName is a filesystem-safe Gradle rootProject.name.
	RootProjectName string
	// PackagePath is PackageID with dots → slashes (Java source dir).
	PackagePath string
	// IntentFiltersXML is extra activity intent-filters from capabilities.
	IntentFiltersXML string
	// Surface is true when the launcher is the native window.
	Surface bool
}

// Options controls Create.
type Options struct {
	// OutDir is the destination project directory (created if missing).
	OutDir string
	// Force overwrites an existing non-empty OutDir.
	Force bool
	// Config is required (PackageID + AppName at minimum after defaults).
	Config Config
}

// Create materializes an Android host project under opts.OutDir.
func Create(opts Options) error {
	cfg, err := normalizeConfig(opts.Config)
	if err != nil {
		return err
	}
	out := strings.TrimSpace(opts.OutDir)
	if out == "" {
		return ErrOutDirRequired
	}
	out, err = filepath.Abs(out)
	if err != nil {
		return err
	}
	if err := common.PrepareOutDir(out, opts.Force); err != nil {
		return err
	}

	if err := cfg.Capabilities.Validate(); err != nil {
		return err
	}
	data := templateData{
		Config:           cfg,
		RootProjectName:  rootProjectName(cfg.PackageID, cfg.AppName),
		PackagePath:      strings.ReplaceAll(cfg.PackageID, ".", "/"),
		IntentFiltersXML: cfg.Capabilities.AndroidIntentFilters(),
		Surface:          cfg.UI == "surface",
	}

	if err := common.WalkTemplateDest(templateFS, data, out, data.javaDest); err != nil {
		return err
	}
	return writeConfigJSON(out, cfg)
}

func (data templateData) javaDest(rel, destRel string) string {
	slash := filepath.ToSlash(rel)
	if strings.HasPrefix(slash, "app/src/main/javagen/") && strings.HasSuffix(rel, ".tmpl") {
		base := strings.TrimSuffix(filepath.Base(rel), ".tmpl")
		return filepath.Join("app", "src", "main", "java", data.PackagePath, base)
	}
	return destRel
}

func normalizeConfig(cfg Config) (Config, error) {
	cfg.PackageID = strings.TrimSpace(cfg.PackageID)
	if err := release.ValidateAppID(cfg.PackageID); err != nil {
		return Config{}, fmt.Errorf("package id: %w", err)
	}
	cfg.AppName = strings.TrimSpace(cfg.AppName)
	if cfg.AppName == "" {
		// Last label of reverse-domain (br.tec.lew.counter → counter).
		parts := strings.Split(cfg.PackageID, ".")
		cfg.AppName = parts[len(parts)-1]
	}
	// Version defaults: prefer goreleaser-style / VCS identity over a fake 0.1.0.
	if cfg.VersionName == "" || cfg.VersionCode <= 0 {
		info := version.Resolve()
		if cfg.VersionName == "" {
			cfg.VersionName = info.AndroidName()
		}
		if cfg.VersionCode <= 0 {
			cfg.VersionCode = version.AndroidCodeFrom(info.Version, 0)
		}
	}
	if strings.TrimSpace(cfg.GoMain) == "" {
		cfg.GoMain = "."
	}
	switch strings.TrimSpace(cfg.UI) {
	case "", "web":
		cfg.UI = ""
	case "surface":
		cfg.UI = "surface"
	default:
		return Config{}, fmt.Errorf("ui %q: want web or surface", cfg.UI)
	}
	return cfg, nil
}

func rootProjectName(packageID, appName string) string {
	// Prefer app name when it is a simple identifier; else last package label.
	name := strings.TrimSpace(appName)
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}, name)
	if name == "" {
		parts := strings.Split(packageID, ".")
		name = parts[len(parts)-1]
	}
	return name
}

func writeConfigJSON(out string, cfg Config) error {
	raw, err := encodeConfigJSON(cfg, "eletrocromo android create")
	if err != nil {
		return err
	}
	// Template already wrote eletrocromo.json; overwrite with canonical JSON.
	return os.WriteFile(filepath.Join(out, ConfigFileName), raw, 0o644)
}
