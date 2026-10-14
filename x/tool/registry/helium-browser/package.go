package pkg

import (
	"context"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

// heliumRepos are the per-OS release repositories. A newer tag in one
// repository often has no build in the others, so latest follows the
// repository for this OS.
var heliumRepos = []struct {
	name string
	goos string
}{
	{"imputnet/helium-linux", "linux"},
	{"imputnet/helium-macos", "darwin"},
	{"imputnet/helium-windows", "windows"},
}

type heliumSource struct {
	goos string
	tool tool.Tool
}

type heliumBrowserTool struct {
	sources []heliumSource
	goos    string
}

func newHeliumBrowser() (tool.Tool, error) {
	sources := make([]heliumSource, 0, len(heliumRepos))
	for _, repo := range heliumRepos {
		inner, err := github.NewTool(repo.name)
		if err != nil {
			return nil, err
		}
		sources = append(sources, heliumSource{goos: repo.goos, tool: inner})
	}
	return &heliumBrowserTool{sources: sources}, nil
}

func (t *heliumBrowserTool) hostOS() string {
	if t.goos != "" {
		return t.goos
	}
	return runtime.GOOS
}

func (t *heliumBrowserTool) ListVersions(ctx context.Context) ([]string, error) {
	host := t.hostOS()
	versions := t.collectVersions(ctx, host)
	if len(versions) == 0 {
		versions = t.collectVersions(ctx, "")
	}
	return install.SortVersionsDesc(versions)
}

func (t *heliumBrowserTool) collectVersions(ctx context.Context, goos string) []string {
	seen := map[string]struct{}{}
	var collected []string
	for _, source := range t.sources {
		if goos != "" && source.goos != goos {
			continue
		}
		vers, err := source.tool.ListVersions(ctx)
		if err != nil {
			continue
		}
		for _, version := range vers {
			version = strings.TrimSpace(version)
			if version == "" {
				continue
			}
			if _, ok := seen[version]; ok {
				continue
			}
			seen[version] = struct{}{}
			collected = append(collected, version)
		}
	}
	return collected
}

func (t *heliumBrowserTool) Install(ctx context.Context, version string, destDir string) error {
	return install.InstallSelectedArtifact(ctx, version, destDir, "helium", "registry:helium-browser", strings.TrimSpace, t.ListVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *heliumBrowserTool) Pin() tool.Pin {
	return tool.Pin{Versioning: "semver"}
}

func (t *heliumBrowserTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	resolved, err := install.ResolveToolVersion(ctx, version, strings.TrimSpace, t.ListVersions)
	if err != nil {
		return nil, err
	}

	var all []tool.Artifact
	for _, source := range t.sources {
		artifactTool, ok := source.tool.(tool.ArtifactTool)
		if !ok {
			continue
		}
		artifacts, err := artifactTool.ListArtifacts(ctx, resolved)
		if err != nil {
			// The version tag may not exist in this OS-specific repo.
			continue
		}
		all = append(all, artifacts...)
	}
	if len(all) == 0 {
		return nil, install.ErrNoPlatformArtifact
	}
	return all, nil
}

func (t *heliumBrowserTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	return install.DefaultInstallArtifact(ctx, artifact, destDir)
}

func (t *heliumBrowserTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("helium"))
}
