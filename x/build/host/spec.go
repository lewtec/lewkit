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

// Spec is the shared app config. Flags overlay eletrocromo.json.
// A non-empty flag replaces that field. An empty flag leaves the file value.
// Config empty probes Dir/eletrocromo.json and continues when that file is absent.
// A non-empty Config is required.
type Spec struct {
	Dir     string
	Config  string
	ID      string
	Name    string
	Version string
	Entry   string
	GoMain  string
	Icon    string
}

// Load reads the JSON file when it exists and merges these fields on top.
// Dir is the app directory used when the file is omitted.
// ctx ends the git reads and an astro or static materialize.
// A non-empty Entry is the --entry flag. It wins over the file and over --main.
// A non-empty GoMain is the --main flag when Entry is empty: it selects go:<flag>.
// A relative static: or astro: flag, a relative --main, and a relative --icon
// use the working directory when that path exists, then the config directory.
// go: and an entry from the file stay relative to the config directory.
// "." as a static: or astro: ref is the config directory.
// The version name and the store integer come from git in the config directory.
// An empty app name is the asset file name, then the last package-id label.
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
	entryFlag := strings.TrimSpace(s.Entry)
	mainFlag := strings.TrimSpace(s.GoMain)
	iconFlag := strings.TrimSpace(s.Icon)
	cfg = apk.Merge(cfg, apk.Config{
		PackageID:   strings.TrimSpace(s.ID),
		AppName:     strings.TrimSpace(s.Name),
		VersionName: strings.TrimSpace(s.Version),
		Entry:       entryFlag,
		GoMain:      mainFlag,
		Icon:        iconFlag,
	})
	if entryFlag != "" {
		cfg.Entry = entryFlag
		cfg.GoMain = ""
	} else if mainFlag != "" {
		cfg.Entry = ""
		cfg.GoMain = mainFlag
	}
	cfg.PackageID = strings.TrimSpace(cfg.PackageID)
	cfg.AppName = strings.TrimSpace(cfg.AppName)
	cfg.VersionName = strings.TrimSpace(cfg.VersionName)
	cfg.Entry = strings.TrimSpace(cfg.Entry)
	cfg.GoMain = strings.TrimSpace(cfg.GoMain)
	cfg.Icon = strings.TrimSpace(cfg.Icon)
	if cfg.Entry == "" && cfg.GoMain == "" {
		cfg.GoMain = dir
	}
	if entryFlag != "" {
		rewritten, err := absoluteAssetEntry(cfg.Entry, base)
		if err != nil {
			return apk.Config{}, "", err
		}
		cfg.Entry = rewritten
	} else if mainFlag != "" {
		main, err := resolveOverlayFile(mainFlag, base)
		if err != nil {
			return apk.Config{}, "", err
		}
		cfg.GoMain = main
	}
	if iconFlag != "" {
		icon, err := resolveOverlayFile(iconFlag, base)
		if err != nil {
			return apk.Config{}, "", err
		}
		cfg.Icon = icon
	}
	fillName(&cfg)
	fillVersion(ctx, &cfg, base)
	main, err := entry.Main(ctx, cfg.Entry, cfg.GoMain, base, cfg.AppName)
	if err != nil {
		return apk.Config{}, "", err
	}
	cfg.GoMain = main
	return cfg, base, nil
}

// absoluteAssetEntry rewrites a flag static: or astro: ref to an absolute path.
// go: and a ref of "." stay as written and resolve against the config directory.
// A parse error is left for entry.Main.
func absoluteAssetEntry(entryText, configBase string) (string, error) {
	parsed, err := entry.ParseEntry(entryText, "")
	if err != nil {
		return entryText, nil
	}
	if (parsed.Name != "static" && parsed.Name != "astro") || parsed.Ref == "." || filepath.IsAbs(parsed.Ref) {
		return entryText, nil
	}
	abs, err := resolveOverlayFile(parsed.Ref, configBase)
	if err != nil {
		return "", err
	}
	return parsed.Name + ":" + abs, nil
}

// resolveOverlayFile locates a path given on a flag.
// An absolute path is cleaned.
// A relative path is the working directory when it exists there,
// then the config directory when it exists there.
// When neither exists the result is the working-directory path.
func resolveOverlayFile(ref, configBase string) (string, error) {
	ref = strings.TrimSpace(ref)
	if filepath.IsAbs(ref) {
		return filepath.Abs(ref)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Abs(filepath.Join(configBase, ref))
	}
	inCwd := filepath.Join(cwd, ref)
	if pathExists(inCwd) {
		return filepath.Abs(inCwd)
	}
	inConfig := filepath.Join(configBase, ref)
	if pathExists(inConfig) {
		return filepath.Abs(inConfig)
	}
	return filepath.Abs(inCwd)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fillName(cfg *apk.Config) {
	if strings.TrimSpace(cfg.AppName) != "" {
		return
	}
	if stem := assetStem(cfg.Entry); stem != "" {
		cfg.AppName = stem
		return
	}
	id := strings.TrimSpace(cfg.PackageID)
	if id == "" {
		return
	}
	parts := strings.Split(id, ".")
	cfg.AppName = parts[len(parts)-1]
}

func assetStem(entryText string) string {
	parsed, err := entry.ParseEntry(entryText, "")
	if err != nil {
		return ""
	}
	if (parsed.Name != "static" && parsed.Name != "astro") || parsed.Ref == "." {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(parsed.Ref), filepath.Ext(parsed.Ref))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
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
