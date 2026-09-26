package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/build/version"
)

// HostConfig is the shared eletrocromo.json identity for iOS and macOS hosts.
type HostConfig struct {
	PackageID    string
	AppName      string
	VersionName  string
	VersionCode  int
	GoMain       string
	Icon         string
	Capabilities Capabilities
}

// ApplyHostDefaults trims identity fields, fills empty app name / version / go_main,
// and validates the package id and capabilities.
func ApplyHostDefaults(cfg HostConfig) (HostConfig, error) {
	cfg.PackageID = strings.TrimSpace(cfg.PackageID)
	if err := release.ValidateAppID(cfg.PackageID); err != nil {
		return HostConfig{}, fmt.Errorf("package id: %w", err)
	}
	cfg.AppName = strings.TrimSpace(cfg.AppName)
	if cfg.AppName == "" {
		parts := strings.Split(cfg.PackageID, ".")
		cfg.AppName = parts[len(parts)-1]
	}
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
	if err := cfg.Capabilities.Validate(); err != nil {
		return HostConfig{}, err
	}
	return cfg, nil
}

// EncodeHostJSON writes the canonical eletrocromo.json body for a host tree.
func EncodeHostJSON(cfg HostConfig, generator string) ([]byte, error) {
	doc := struct {
		SchemaVersion int          `json:"schema_version"`
		PackageID     string       `json:"package_id"`
		AppName       string       `json:"app_name"`
		VersionName   string       `json:"version_name"`
		VersionCode   int          `json:"version_code"`
		GoMain        string       `json:"go_main"`
		Icon          string       `json:"icon,omitempty"`
		Generator     string       `json:"generator"`
		Capabilities  Capabilities `json:"capabilities,omitempty"`
	}{
		SchemaVersion: 1,
		PackageID:     cfg.PackageID,
		AppName:       cfg.AppName,
		VersionName:   cfg.VersionName,
		VersionCode:   cfg.VersionCode,
		GoMain:        cfg.GoMain,
		Icon:          cfg.Icon,
		Generator:     generator,
		Capabilities:  cfg.Capabilities,
	}
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}
