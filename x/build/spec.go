package build

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/apk"
)

// Spec is the shared app config. Flags overlay eletrocromo.json. Empty flags
// leave the file values in place.
type Spec struct {
	Dir     string
	Config  string
	ID      string
	Name    string
	Version string
	GoMain  string
}

// Load reads the JSON file when it exists and merges these fields on top.
// Dir is the module directory used when the file is omitted.
func (s Spec) Load() (apk.Config, string, error) {
	dir := strings.TrimSpace(s.Dir)
	if dir == "" {
		dir = "."
	}
	cfg := apk.Config{}
	base := dir
	path := strings.TrimSpace(s.Config)
	if path == "" {
		try := filepath.Join(dir, apk.ConfigFileName)
		info, err := os.Stat(try)
		if err == nil && !info.IsDir() {
			path = try
		}
	}
	if path != "" {
		loaded, loadedBase, err := apk.LoadConfig(path)
		if err != nil {
			return apk.Config{}, "", err
		}
		cfg = loaded
		base = loadedBase
	}
	cfg = apk.Merge(cfg, apk.Config{
		PackageID:   s.ID,
		AppName:     s.Name,
		VersionName: s.Version,
		GoMain:      s.GoMain,
	})
	if strings.TrimSpace(cfg.GoMain) == "" {
		cfg.GoMain = dir
	}
	main, err := apk.ResolveGoMain(cfg.GoMain, base)
	if err != nil {
		return apk.Config{}, "", err
	}
	cfg.GoMain = main
	return cfg, base, nil
}
