package applications

import (
	"context"
	"fmt"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
	"path/filepath"
	"runtime"
	"strings"
)

func init() {
	registry.RegisterTool("flutter", newFlutter)
}

type flutterTool struct{}

func newFlutter() (tool.Tool, error) {
	return &flutterTool{}, nil
}

func (t *flutterTool) ListVersions(ctx context.Context) ([]string, error) {
	return t.listVersions(ctx)
}

func (t *flutterTool) Install(ctx context.Context, version string, destDir string) error {
	return installFirstArtifact(ctx, version, destDir, normalizeFlutterVersion, t.listVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *flutterTool) Pin() tool.Pin {
	var pin tool.Pin
	// Flutter releases come from Google Cloud Storage; no standard Renovate
	// datasource matches the official release index + shas.

	return pin
}

func (t *flutterTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	v, err := resolveToolVersion(ctx, version, normalizeFlutterVersion, t.listVersions)
	if err != nil {
		return nil, err
	}

	idx, err := t.fetchReleasesIndex(ctx)
	if err != nil {
		return nil, err
	}

	for _, r := range idx.Releases {
		if r.Version != v {
			continue
		}
		if !t.archiveMatchesPlatform(r.Archive) {
			continue
		}
		base := strings.TrimRight(idx.BaseURL, "/")
		fullURL := base + "/" + strings.TrimLeft(r.Archive, "/")
		hash := ""
		if r.SHA256 != "" {
			hash = "sha256:" + r.SHA256
		}
		return []tool.Artifact{{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
			URL:  fullURL,
			Hash: hash,
		}}, nil
	}
	return nil, ErrNoPlatformArtifact
}

func (t *flutterTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	return defaultInstallArtifact(ctx, artifact, destDir)
}

func (t *flutterTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	return ensureToolBinary(ctx, version, cmdName, destDir, "Flutter", t.Install)
}

// --- helpers ---

type flutterReleasesIndex struct {
	BaseURL  string           `json:"base_url"`
	Releases []flutterRelease `json:"releases"`
}

type flutterRelease struct {
	Hash    string `json:"hash"`
	Channel string `json:"channel"`
	Version string `json:"version"`
	Archive string `json:"archive"`
	SHA256  string `json:"sha256"`
}

func (t *flutterTool) fetchReleasesIndex(ctx context.Context) (flutterReleasesIndex, error) {
	var idx flutterReleasesIndex
	if err := getJSON(ctx, t.releasesURL(), &idx); err != nil {
		return flutterReleasesIndex{}, err
	}
	return idx, nil
}

func (t *flutterTool) listVersions(ctx context.Context) ([]string, error) {
	idx, err := t.fetchReleasesIndex(ctx)
	if err != nil {
		return nil, err
	}

	out := []string{}
	seen := map[string]bool{}
	for _, r := range idx.Releases {
		ver := strings.TrimSpace(r.Version)
		if ver == "" || seen[ver] {
			continue
		}
		if t.archiveMatchesPlatform(r.Archive) {
			seen[ver] = true
			out = append(out, ver)
		}
	}
	if len(out) == 0 {
		return nil, ErrNoVersions
	}
	return out, nil
}

func (t *flutterTool) releasesURL() string {
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	return fmt.Sprintf("https://storage.googleapis.com/flutter_infra_release/releases/releases_%s.json", platform)
}

func (t *flutterTool) archiveMatchesPlatform(archive string) bool {
	if archive == "" {
		return false
	}
	base := filepath.Base(archive)
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	switch goos {
	case "linux":
		// The linux index primarily lists x64 entries; arm64 linux archives use
		// a flutter_linux_arm64_... name but are not separately enumerated in
		// the index today. Accept any linux_ entry so versions list and resolve
		// on linux (arm64 linux will currently receive the x64 SDK tarball).
		return strings.HasPrefix(base, "flutter_linux_")
	case "darwin":
		if goarch == "arm64" {
			return strings.Contains(base, "macos_arm64_")
		}
		return strings.HasPrefix(base, "flutter_macos_") && !strings.Contains(base, "_arm64_")
	case "windows":
		if goarch == "arm64" {
			return strings.Contains(base, "windows_arm64_")
		}
		return strings.HasPrefix(base, "flutter_windows_") && !strings.Contains(base, "_arm64_")
	default:
		return false
	}
}

func normalizeFlutterVersion(version string) string {
	return normalizeVPrefixed(version)
}

func (t *flutterTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("flutter"))
}
