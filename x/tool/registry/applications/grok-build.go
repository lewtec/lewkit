package applications

import (
	"context"
	"errors"
	"fmt"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var ErrGrokBuildProbeFailure = errors.New("probe grok-build latest from x.ai channels")

func init() {
	registry.RegisterTool("grok-build", newGrokBuild)
}

type grokBuildTool struct{}

func newGrokBuild() (tool.Tool, error) {
	return &grokBuildTool{}, nil
}

func (t *grokBuildTool) ListVersions(ctx context.Context) ([]string, error) {
	v, err := t.probeLatest(ctx)
	if err != nil {
		return nil, err
	}
	return []string{v}, nil
}

func (t *grokBuildTool) Install(ctx context.Context, version string, destDir string) error {
	arts, err := t.ListArtifacts(ctx, version)
	if err != nil {
		return err
	}
	return t.InstallArtifact(ctx, arts[0], destDir)
}

func (t *grokBuildTool) Pin() tool.Pin {
	var pin tool.Pin
	// no renovate datasource metadata for this internal tool

	return pin
}

func (t *grokBuildTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	v, err := t.resolveVersion(ctx, version)
	if err != nil {
		return nil, err
	}
	return []tool.Artifact{t.artifactFor(v)}, nil
}

// resolveVersion trims version and probes the stable channel when empty or "latest".
func (t *grokBuildTool) resolveVersion(ctx context.Context, version string) (string, error) {
	v := strings.TrimSpace(version)
	if v == "" || v == "latest" {
		return t.probeLatest(ctx)
	}
	return v, nil
}

// artifactFor builds the current-platform download artifact for a concrete version.
func (t *grokBuildTool) artifactFor(version string) tool.Artifact {
	u := "https://x.ai/cli/grok-" + version + "-" + grokPlatform(runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		u += ".exe"
	}
	return tool.Artifact{URL: u, OS: runtime.GOOS, Arch: runtime.GOARCH}
}

func (t *grokBuildTool) InstallArtifact(ctx context.Context, art tool.Artifact, destDir string) error {
	bin := "grok"
	if runtime.GOOS == "windows" {
		bin = "grok.exe"
	}
	path := filepath.Join(destDir, bin)

	urls := []string{art.URL}
	fallback := strings.Replace(art.URL, "https://x.ai/cli/", "https://storage.googleapis.com/grok-build-public-artifacts/cli/", 1)
	if fallback != art.URL {
		urls = append(urls, fallback)
	}
	if err := tool.DownloadFirst(ctx, urls, path, tool.DownloadOptions{Mode: 0o755}); err != nil {
		return err
	}

	command := exec.CommandContext(ctx, path, "--version")
	command.Stdin = strings.NewReader("")
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			slog.WarnContext(ctx, "remove broken binary", "error", removeErr, "path", path)
		}
		return fmt.Errorf("smoke test failed: %w", err)
	}

	agent := "agent"
	if runtime.GOOS == "windows" {
		agent = "agent.exe"
	}
	if rmErr := os.Remove(filepath.Join(destDir, agent)); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
		// agent symlink target removal is best-effort (may not exist)
	}
	if err := os.Symlink(bin, filepath.Join(destDir, agent)); err != nil {
		return fmt.Errorf("symlink agent: %w", err)
	}
	return nil
}

func (t *grokBuildTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	return tool.EnsureBinary(destDir, cmdName, "grok", func() error {
		return t.Install(ctx, version, destDir)
	}, "grok")
}

// --- grok-specific bits (follows https://x.ai/cli/install.sh artifact layout) ---

func (t *grokBuildTool) probeLatest(ctx context.Context) (string, error) {
	for _, base := range []string{
		"https://x.ai/cli",
		"https://storage.googleapis.com/grok-build-public-artifacts/cli",
	} {
		b, err := getBytes(ctx, base+"/stable")
		if err != nil {
			continue
		}
		if s := strings.TrimSpace(string(b)); s != "" {
			return s, nil
		}
	}
	return "", ErrGrokBuildProbeFailure
}

// grokPlatform maps GOOS/GOARCH to the x.ai CLI artifact suffix
// (e.g. linux-x86_64, macos-aarch64). Android has no dedicated build;
// use the Linux binary (same ABI as Termux and other Android Linux userlands).
func grokPlatform(goos, goarch string) string {
	osn := goos
	switch osn {
	case "darwin":
		osn = "macos"
	case "android":
		osn = "linux"
	}
	arch := goarch
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	}
	return osn + "-" + arch
}

func (t *grokBuildTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("grok"))
}
