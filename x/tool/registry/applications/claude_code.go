package applications

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
	"os"
	"runtime"
	"strings"
)

const claudeCodeReleasesBaseURL = "https://downloads.claude.ai/claude-code-releases"

func init() {
	registry.RegisterTool("claude-code", newClaudeCode)
}

type claudeCodeTool struct {
	baseURL  string
	fetchURL func(context.Context, string) ([]byte, error)
}

type claudeCodeManifest struct {
	Version   string                            `json:"version"`
	Platforms map[string]claudeCodePlatformInfo `json:"platforms"`
}

type claudeCodePlatformInfo struct {
	Binary   string `json:"binary"`
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
}

func newClaudeCode() (tool.Tool, error) {
	return &claudeCodeTool{baseURL: claudeCodeReleasesBaseURL}, nil
}

func (t *claudeCodeTool) ListVersions(ctx context.Context) ([]string, error) {
	latest, err := t.ResolveVersion(ctx, "latest")
	if err != nil {
		return nil, err
	}

	stable, err := t.ResolveVersion(ctx, "stable")
	if err != nil {
		return nil, err
	}

	versions := []string{latest}
	if stable != "" && stable != latest {
		versions = append(versions, stable)
	}
	return versions, nil
}

func (t *claudeCodeTool) ResolveVersion(ctx context.Context, version string) (string, error) {
	requested := strings.TrimSpace(version)
	if requested == "" {
		requested = "latest"
	}

	switch requested {
	case "latest", "stable":
		body, err := t.fetch(ctx, fmt.Sprintf("%s/%s", t.baseURL, requested))
		if err != nil {
			return "", err
		}
		resolved := strings.TrimSpace(string(body))
		if resolved == "" {
			return "", fmt.Errorf("empty claude-code version response for %s", requested)
		}
		return resolved, nil
	default:
		return requested, nil
	}
}

func (t *claudeCodeTool) Install(ctx context.Context, version string, destDir string) error {
	artifacts, err := t.ListArtifacts(ctx, version)
	if err != nil {
		return err
	}

	artifact := tool.SelectArtifact(artifacts, runtime.GOOS, runtime.GOARCH, "claude")
	if artifact == nil {
		return fmt.Errorf("no suitable artifact found for %s/%s for registry:claude-code@%s", runtime.GOOS, runtime.GOARCH, version)
	}

	return t.InstallArtifact(ctx, *artifact, destDir)
}

func (t *claudeCodeTool) Pin() tool.Pin {
	var pin tool.Pin
	pin.Versioning = "semver"

	return pin
}

func (t *claudeCodeTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	resolvedVersion, err := t.ResolveVersion(ctx, version)
	if err != nil {
		return nil, err
	}

	manifest, err := t.fetchManifest(ctx, resolvedVersion)
	if err != nil {
		return nil, err
	}

	platformKey := t.currentPlatform()
	platformInfo, ok := manifest.Platforms[platformKey]
	if !ok {
		return nil, fmt.Errorf("platform %s not found in claude-code manifest for %s", platformKey, resolvedVersion)
	}

	url := fmt.Sprintf("%s/%s/%s/%s", t.baseURL, resolvedVersion, platformKey, platformInfo.Binary)
	return []tool.Artifact{{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		URL:  url,
		Hash: "sha256:" + platformInfo.Checksum,
		Size: platformInfo.Size,
	}}, nil
}

func (t *claudeCodeTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	return tool.InstallArtifact(ctx, artifact, destDir, tool.DownloadOptions{Mode: 0o755})
}

func (t *claudeCodeTool) fetchManifest(ctx context.Context, version string) (claudeCodeManifest, error) {
	body, err := t.fetch(ctx, fmt.Sprintf("%s/%s/manifest.json", t.baseURL, version))
	if err != nil {
		return claudeCodeManifest{}, err
	}

	var manifest claudeCodeManifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return claudeCodeManifest{}, err
	}
	return manifest, nil
}

func (t *claudeCodeTool) fetch(ctx context.Context, url string) ([]byte, error) {
	if t.fetchURL != nil {
		return t.fetchURL(ctx, url)
	}
	return getBytes(ctx, url)
}

func (t *claudeCodeTool) currentPlatform() string {
	osName := runtime.GOOS
	if osName == "windows" {
		osName = "win32"
	}

	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x64"
	case "386":
		arch = "x86"
	}

	if osName == "linux" && t.isMusl() {
		return osName + "-" + arch + "-musl"
	}
	return osName + "-" + arch
}

func (t *claudeCodeTool) isMusl() bool {
	for _, path := range []string{
		"/lib/libc.musl-x86_64.so.1",
		"/lib/libc.musl-aarch64.so.1",
		"/lib/ld-musl-x86_64.so.1",
		"/lib/ld-musl-aarch64.so.1",
	} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func (t *claudeCodeTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("claude"))
}
