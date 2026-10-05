package apk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

// Config / go_main validation sentinels.
var (
	ErrConfigPathEmpty = errors.New("config path is empty")
	ErrGoMainNotDir    = errors.New("go_main must be a directory (main package)")
)

// ConfigFileName is the standard project config next to the Go app (or in a generated host).
const ConfigFileName = "eletrocromo.json"

// DefaultABIs for the packaged APK.
//
// With CGO_ENABLED=0 only android/arm64 links today (no NDK). Prefer arm64-v8a
// for pure-Go apps; add other ABIs when you have an NDK toolchain and cgo.
var DefaultABIs = []string{"arm64-v8a"}

// abiToGOARCH maps Android ABI → Go GOARCH (CGO_ENABLED=0 GOOS=android).
var abiToGOARCH = map[string]string{
	"arm64-v8a":   "arm64",
	"armeabi-v7a": "arm",
	"x86_64":      "amd64",
	"x86":         "386",
}

// fileConfig is the on-disk JSON shape (schema_version included).
type fileConfig struct {
	SchemaVersion int             `json:"schema_version"`
	PackageID     string          `json:"package_id"`
	AppName       string          `json:"app_name"`
	VersionName   string          `json:"version_name"`
	VersionCode   int             `json:"version_code"`
	GoMain        string          `json:"go_main"`
	Icon          string          `json:"icon,omitempty"`
	Generator     string          `json:"generator,omitempty"`
	ABIs          []string        `json:"abis,omitempty"`
	Capabilities  json.RawMessage `json:"capabilities,omitempty"`
}

// LoadConfig reads eletrocromo.json from path (file or directory containing it).
// BaseDir is the directory used to resolve relative go_main.
func LoadConfig(path string) (cfg Config, baseDir string, err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Config{}, "", ErrConfigPathEmpty
	}
	st, err := os.Stat(path)
	if err != nil {
		return Config{}, "", err
	}
	var file string
	if st.IsDir() {
		baseDir = path
		file = filepath.Join(path, ConfigFileName)
	} else {
		file = path
		baseDir = filepath.Dir(path)
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		return Config{}, "", err
	}
	var doc fileConfig
	if err := json.Unmarshal(raw, &doc); err != nil {
		return Config{}, "", fmt.Errorf("%s: %w", file, err)
	}
	cfg = Config{
		PackageID:   doc.PackageID,
		AppName:     doc.AppName,
		VersionName: doc.VersionName,
		VersionCode: doc.VersionCode,
		GoMain:      doc.GoMain,
		Icon:        doc.Icon,
		ABIs:        doc.ABIs,
	}
	caps, err := common.ParseCapabilities(doc.Capabilities)
	if err != nil {
		return Config{}, "", fmt.Errorf("%s: %w", file, err)
	}
	cfg.Capabilities = caps
	return cfg, baseDir, nil
}

// Merge overlays non-zero flag values onto base (flags win).
func Merge(base, overlay Config) Config {
	out := base
	if strings.TrimSpace(overlay.PackageID) != "" {
		out.PackageID = overlay.PackageID
	}
	if strings.TrimSpace(overlay.AppName) != "" {
		out.AppName = overlay.AppName
	}
	if strings.TrimSpace(overlay.VersionName) != "" {
		out.VersionName = overlay.VersionName
	}
	if overlay.VersionCode > 0 {
		out.VersionCode = overlay.VersionCode
	}
	if strings.TrimSpace(overlay.GoMain) != "" {
		out.GoMain = overlay.GoMain
	}
	if strings.TrimSpace(overlay.Icon) != "" {
		out.Icon = overlay.Icon
	}
	if len(overlay.ABIs) > 0 {
		out.ABIs = append([]string(nil), overlay.ABIs...)
	}
	if !overlay.Capabilities.Empty() {
		out.Capabilities = overlay.Capabilities
	}
	return out
}

// ResolveGoMain returns an absolute directory containing the Go main package.
// Relative paths are resolved against baseDir (config file directory, or cwd).
func ResolveGoMain(goMain, baseDir string) (string, error) {
	goMain = strings.TrimSpace(goMain)
	if goMain == "" {
		goMain = "."
	}
	if !filepath.IsAbs(goMain) {
		goMain = filepath.Join(baseDir, goMain)
	}
	abs, err := filepath.Abs(goMain)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("go_main %q: %w", abs, err)
	}
	if !st.IsDir() {
		return "", fmt.Errorf("%w: %s", ErrGoMainNotDir, abs)
	}
	return abs, nil
}

func (c Config) withDefaults(ctx context.Context) (Config, error) {
	return normalizeConfig(ctx, c)
}

func (c Config) abis() []string {
	if len(c.ABIs) > 0 {
		return c.ABIs
	}
	return append([]string(nil), DefaultABIs...)
}

func encodeConfigJSON(cfg Config, generator string) ([]byte, error) {
	abis := cfg.abis()
	doc := fileConfig{
		SchemaVersion: 1,
		PackageID:     cfg.PackageID,
		AppName:       cfg.AppName,
		VersionName:   cfg.VersionName,
		VersionCode:   cfg.VersionCode,
		GoMain:        cfg.GoMain,
		Icon:          cfg.Icon,
		Generator:     generator,
		ABIs:          abis,
	}
	if !cfg.Capabilities.Empty() {
		rawCaps, err := json.Marshal(cfg.Capabilities)
		if err != nil {
			return nil, err
		}
		doc.Capabilities = rawCaps
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
