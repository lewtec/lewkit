package host

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	envData   = "LEWKIT_DATA_DIR"
	envCache  = "LEWKIT_CACHE_DIR"
	envConfig = "LEWKIT_CONFIG_DIR"
)

func paths(
	appID, goos string,
	getenv func(string) string,
	homeFn, cacheFn, configFn func() (string, error),
) (layout, error) {
	data, err := pick(getenv(envData), func() (string, error) {
		base, err := dataHome(goos, getenv, homeFn)
		if err != nil {
			return "", err
		}
		return filepath.Join(base, appID), nil
	})
	if err != nil {
		return layout{}, fmt.Errorf("data dir: %w", err)
	}
	cache, err := pick(getenv(envCache), func() (string, error) {
		base, err := cacheFn()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, appID), nil
	})
	if err != nil {
		return layout{}, fmt.Errorf("cache dir: %w", err)
	}
	config, err := pick(getenv(envConfig), func() (string, error) {
		base, err := configFn()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, appID), nil
	})
	if err != nil {
		return layout{}, fmt.Errorf("config dir: %w", err)
	}
	return layout{
		Data:   data,
		Cache:  cache,
		Config: config,
		Inbox:  filepath.Join(cache, "inbox"),
	}, nil
}

type layout struct {
	Data, Cache, Config, Inbox string
}

func pick(env string, fallback func() (string, error)) (string, error) {
	if v := strings.TrimSpace(env); v != "" {
		return v, nil
	}
	return fallback()
}

func dataHome(goos string, getenv func(string) string, homeFn func() (string, error)) (string, error) {
	if v := getenv("XDG_DATA_HOME"); v != "" {
		return v, nil
	}
	home, err := homeFn()
	if err != nil {
		return "", err
	}
	switch goos {
	case "windows":
		if v := getenv("LOCALAPPDATA"); v != "" {
			return v, nil
		}
		return filepath.Join(home, "AppData", "Local"), nil
	case "darwin":
		return filepath.Join(home, "Library", "Application Support"), nil
	default:
		return filepath.Join(home, ".local", "share"), nil
	}
}

func livePaths(appID string) (layout, error) {
	return paths(appID, runtime.GOOS, os.Getenv, os.UserHomeDir, os.UserCacheDir, os.UserConfigDir)
}
