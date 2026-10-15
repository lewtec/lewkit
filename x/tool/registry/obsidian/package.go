package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/httpclient"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

const obsidianRepo = "obsidianmd/obsidian-releases"

var obsidianVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

type obsidianTool struct {
	inner    tool.Tool
	fetchURL func(context.Context, string) ([]byte, error)
}

type obsidianAsset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
	APIURL             string `json:"url"`
}

func newObsidian() (tool.Tool, error) {
	inner, err := github.NewTool(obsidianRepo, "obsidian")
	if err != nil {
		return nil, err
	}
	return &obsidianTool{inner: inner}, nil
}

func (t *obsidianTool) ListVersions(ctx context.Context) ([]string, error) {
	versions, err := t.inner.ListVersions(ctx)
	if err != nil {
		return nil, err
	}
	return install.SortVersionsDesc(install.KeepVersions(versions, obsidianVersion))
}

func (t *obsidianTool) Install(ctx context.Context, version, destDir string) error {
	return install.InstallSelectedArtifact(ctx, version, destDir, "obsidian", "registry:obsidian", normalizeObsidianVersion, t.ListVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *obsidianTool) Pin() tool.Pin {
	return tool.InheritSemver(t.inner)
}

func (t *obsidianTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	resolved, err := install.ResolveToolVersion(ctx, version, normalizeObsidianVersion, t.ListVersions)
	if err != nil {
		return nil, err
	}
	body, err := t.fetch(ctx, obsidianReleaseURL(resolved))
	if err != nil {
		return nil, err
	}
	assets, err := decodeObsidianAssets(body)
	if err != nil {
		return nil, err
	}
	artifacts := obsidianArtifacts(assets)
	if len(artifacts) == 0 {
		return nil, ErrNoPlatformArtifact
	}
	return artifacts, nil
}

func (t *obsidianTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	if artifactTool, ok := t.inner.(tool.ArtifactTool); ok {
		return artifactTool.InstallArtifact(ctx, artifact, destDir)
	}
	return install.DefaultInstallArtifact(ctx, artifact, destDir)
}

func (t *obsidianTool) EnsureBinary(ctx context.Context, version, cmdName, destDir string) (string, error) {
	return install.EnsureToolBinary(ctx, version, cmdName, destDir, "Obsidian", t.Install)
}

func (t *obsidianTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("obsidian"))
}

func (t *obsidianTool) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	if t.fetchURL != nil {
		return t.fetchURL(ctx, rawURL)
	}
	request, err := github.NewAPIRequest(ctx, http.MethodGet, rawURL)
	if err != nil {
		return nil, err
	}
	httpDriver, err := driver.Get[httpclient.Driver](ctx)
	if err != nil {
		return nil, err
	}
	response, err := httpDriver.Client().Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", github.ErrAPIError, response.Status)
	}
	return body, nil
}

func obsidianReleaseURL(version string) string {
	return "https://api.github.com/repos/" + obsidianRepo + "/releases/tags/v" + version
}

func normalizeObsidianVersion(version string) string {
	return install.NormalizeVPrefixed(version)
}

func obsidianVersion(tag string) (string, bool) {
	version := normalizeObsidianVersion(tag)
	if !obsidianVersionPattern.MatchString(version) {
		return "", false
	}
	return version, true
}

func decodeObsidianAssets(body []byte) ([]obsidianAsset, error) {
	var release struct {
		Assets []obsidianAsset `json:"assets"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecodeRelease, err)
	}
	return release.Assets, nil
}

func obsidianArtifacts(assets []obsidianAsset) []tool.Artifact {
	var artifacts []tool.Artifact
	for _, asset := range assets {
		for _, platform := range obsidianPlatforms(asset.Name) {
			artifacts = append(artifacts, tool.Artifact{
				OS:                platform.osName,
				Arch:              platform.arch,
				URL:               asset.BrowserDownloadURL,
				Hash:              obsidianHash(asset.Digest),
				Size:              asset.Size,
				GitHubAssetID:     asset.ID,
				GitHubAssetAPIURL: asset.APIURL,
			})
		}
	}
	return artifacts
}

type obsidianPlatform struct {
	osName string
	arch   string
}

// obsidianPlatforms maps one release asset onto Go OS and arch values.
// A dmg without an architecture is one build for both darwin arches.
// Linux archives without an architecture are amd64. arm64 is named in the file.
func obsidianPlatforms(name string) []obsidianPlatform {
	base := strings.ToLower(path.Base(name))
	switch {
	case strings.Contains(base, ".asar"),
		strings.HasSuffix(base, ".deb"),
		strings.HasSuffix(base, ".rpm"),
		strings.HasSuffix(base, ".apk"),
		strings.HasSuffix(base, ".snap"),
		strings.HasSuffix(base, ".zsync"):
		return nil
	}

	arch := ""
	switch {
	case strings.Contains(base, "arm64"), strings.Contains(base, "aarch64"):
		arch = "arm64"
	case strings.Contains(base, "amd64"), strings.Contains(base, "x86_64"), strings.Contains(base, "x64"):
		arch = "amd64"
	}

	switch {
	case strings.HasSuffix(base, ".appimage"):
		if arch == "" {
			arch = "amd64"
		}
		return []obsidianPlatform{{osName: "linux", arch: arch}}
	case strings.HasSuffix(base, ".tar.gz"), strings.HasSuffix(base, ".tgz"), strings.HasSuffix(base, ".tar.xz"):
		if arch == "" {
			arch = "amd64"
		}
		return []obsidianPlatform{{osName: "linux", arch: arch}}
	case strings.HasSuffix(base, ".dmg"):
		if arch == "" {
			return []obsidianPlatform{{osName: "darwin", arch: "amd64"}, {osName: "darwin", arch: "arm64"}}
		}
		return []obsidianPlatform{{osName: "darwin", arch: arch}}
	case strings.HasSuffix(base, ".exe"):
		if arch == "" {
			arch = "amd64"
		}
		return []obsidianPlatform{{osName: "windows", arch: arch}}
	default:
		return nil
	}
}

func obsidianHash(digest string) string {
	digest = strings.TrimSpace(digest)
	if digest == "" {
		return ""
	}
	if strings.Contains(digest, ":") {
		return digest
	}
	return "sha256:" + digest
}
