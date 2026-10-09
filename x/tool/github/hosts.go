package github

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go.yaml.in/yaml/v3"
)

type ghHost struct {
	User       string            `yaml:"user"`
	OAuthToken string            `yaml:"oauth_token"`
	Users      map[string]ghUser `yaml:"users"`
}

type ghUser struct {
	OAuthToken string `yaml:"oauth_token"`
}

// tokenFromHosts reads the plaintext oauth_token gh writes when secure
// storage is unavailable. Host-level oauth_token is the active token.
// users.<user>.oauth_token is the copy kept for that account.
func tokenFromHosts(context.Context) (string, error) {
	host, err := loadGitHubHost()
	if err != nil || host == nil {
		return "", err
	}
	if token := cleanToken(host.OAuthToken); token != "" {
		return token, nil
	}
	user := strings.TrimSpace(host.User)
	if user == "" {
		return "", nil
	}
	return cleanToken(host.Users[user].OAuthToken), nil
}

func githubHostUser() string {
	host, err := loadGitHubHost()
	if err != nil || host == nil {
		return ""
	}
	return strings.TrimSpace(host.User)
}

func loadGitHubHost() (*ghHost, error) {
	path := ghHostsPath()
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var hosts map[string]ghHost
	if err := yaml.Unmarshal(data, &hosts); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	host, ok := hosts[githubHost]
	if !ok {
		return nil, nil
	}
	return &host, nil
}

// ghHostsPath follows go-gh: GH_CONFIG_DIR, then XDG_CONFIG_HOME/gh,
// then %AppData%/GitHub CLI on Windows, then ~/.config/gh.
func ghHostsPath() string {
	dir := ghConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "hosts.yml")
}

func ghConfigDir() string {
	if dir := strings.TrimSpace(os.Getenv("GH_CONFIG_DIR")); dir != "" {
		return dir
	}
	if dir := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); dir != "" {
		return filepath.Join(dir, "gh")
	}
	if runtime.GOOS == "windows" {
		if dir := strings.TrimSpace(os.Getenv("AppData")); dir != "" {
			return filepath.Join(dir, "GitHub CLI")
		}
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".config", "gh")
}
