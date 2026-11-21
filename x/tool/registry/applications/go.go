package applications

import (
	"context"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
	"runtime"
	"strings"
)

func init() {
	registry.RegisterTool("golang", newGo)
}

type goTool struct{}

func newGo() (tool.Tool, error) {
	return &goTool{}, nil
}

func (t *goTool) ListVersions(ctx context.Context) ([]string, error) {
	return t.listVersions(ctx)
}

func (t *goTool) Install(ctx context.Context, version string, destDir string) error {
	return installFirstArtifact(ctx, version, destDir, normalizeGoVersion, t.listVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *goTool) Pin() tool.Pin {
	var pin tool.Pin
	// No standard renovate datasource for the custom go.dev tarballs.

	return pin
}

func (t *goTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	relVer, err := resolveToolVersion(ctx, version, goVersionForIndex, t.listVersions)
	if err != nil {
		return nil, err
	}

	releases, err := fetchGoReleases(ctx)
	if err != nil {
		return nil, err
	}

	osName := runtime.GOOS
	archName := runtime.GOARCH

	for _, r := range releases {
		if r.Version != relVer {
			continue
		}
		for _, f := range r.Files {
			if f.Kind != "archive" {
				continue
			}
			if f.OS == osName && f.Arch == archName {
				return []tool.Artifact{{
					OS:   f.OS,
					Arch: f.Arch,
					URL:  "https://go.dev/dl/" + f.Filename,
					Hash: "sha256:" + f.SHA256,
					Size: f.Size,
				}}, nil
			}
		}
	}
	return nil, ErrNoPlatformArtifact
}

func (t *goTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	return defaultInstallArtifact(ctx, artifact, destDir)
}

func (t *goTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	// After StripTopLevelDir the Go tarball/zip layout gives us bin/go, bin/gofmt, etc.
	return ensureToolBinary(ctx, version, cmdName, destDir, "Go", t.Install)
}

// --- helpers ---

type goRelease struct {
	Version string   `json:"version"`
	Stable  bool     `json:"stable"`
	Files   []goFile `json:"files"`
}

type goFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
}

func fetchGoReleases(ctx context.Context) ([]goRelease, error) {
	var releases []goRelease
	if err := getJSON(ctx, "https://go.dev/dl/?mode=json", &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func (t *goTool) listVersions(ctx context.Context) ([]string, error) {
	releases, err := fetchGoReleases(ctx)
	if err != nil {
		return nil, err
	}

	out := []string{}
	seen := map[string]bool{}
	osName := runtime.GOOS
	archName := runtime.GOARCH
	for _, r := range releases {
		for _, f := range r.Files {
			if f.Kind == "archive" && f.OS == osName && f.Arch == archName {
				ver := strings.TrimPrefix(r.Version, "go")
				if ver != "" && !seen[ver] {
					seen[ver] = true
					out = append(out, ver)
				}
				break
			}
		}
	}
	return out, nil
}

func normalizeGoVersion(version string) string {
	return normalizeVersion(version, "go", "v")
}

func goVersionForIndex(version string) string {
	v := normalizeGoVersion(version)
	if v == "" || v == "latest" {
		return v
	}
	return "go" + v
}

func (t *goTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("go"))
}
