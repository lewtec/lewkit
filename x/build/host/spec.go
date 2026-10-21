package host

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/entry"
	_ "github.com/lewtec/lewkit/x/build/entry/prelude"
	"github.com/lewtec/lewkit/x/build/gen/apk"
	"github.com/lewtec/lewkit/x/build/version"
)

// Spec is the shared app config. Flags overlay eletrocromo.json. Empty flags
// leave the file values in place.
type Spec struct {
	Dir     string
	Config  string
	ID      string
	Name    string
	Version string
	Entry   string
	GoMain  string
}

// Load reads the JSON file when it exists and merges these fields on top.
// Dir is the module directory used when the file is omitted.
// ctx ends the git reads and an astro or static materialize.
// A non-empty Entry is the --entry flag. It wins over the file and over --main.
// A non-empty GoMain is the --main flag when Entry is empty: it selects go:<flag>.
// The version name and the store integer come from git in the config directory.
func (s Spec) Load(ctx context.Context) (apk.Config, string, error) {
	if err := ctx.Err(); err != nil {
		return apk.Config{}, "", context.Cause(ctx)
	}
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
		Entry:       s.Entry,
		GoMain:      s.GoMain,
	})
	if strings.TrimSpace(s.Entry) != "" {
		cfg.Entry = strings.TrimSpace(s.Entry)
		cfg.GoMain = ""
	} else if strings.TrimSpace(s.GoMain) != "" {
		cfg.Entry = ""
		cfg.GoMain = s.GoMain
	}
	if strings.TrimSpace(cfg.Entry) == "" && strings.TrimSpace(cfg.GoMain) == "" {
		cfg.GoMain = dir
	}
	fillVersion(ctx, &cfg, base)
	main, err := entry.Main(ctx, cfg.Entry, cfg.GoMain, base, cfg.AppName)
	if err != nil {
		return apk.Config{}, "", err
	}
	cfg.GoMain = main
	return cfg, base, nil
}

func fillVersion(ctx context.Context, cfg *apk.Config, base string) {
	if strings.TrimSpace(cfg.VersionName) == "" {
		cfg.VersionName = version.ResolveTree(ctx, base).AndroidName()
	}
	if cfg.VersionCode <= 0 {
		cfg.VersionCode = version.GitCommitCount(ctx, base)
		if cfg.VersionCode < 1 {
			cfg.VersionCode = 1
		}
	}
}
