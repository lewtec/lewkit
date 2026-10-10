package pkg

import (
	"context"
	"strings"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

type heliumBrowserTool struct {
	tools []tool.Tool
}

func newHeliumBrowser() (tool.Tool, error) {
	repoNames := []string{
		"imputnet/helium-linux",
		"imputnet/helium-macos",
		"imputnet/helium-windows",
	}
	tools := make([]tool.Tool, 0, len(repoNames))
	for _, r := range repoNames {
		t, err := github.NewTool(r)
		if err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return &heliumBrowserTool{tools: tools}, nil
}

func (t *heliumBrowserTool) ListVersions(ctx context.Context) ([]string, error) {
	seen := make(map[string]struct{})
	var collected []string
	for _, inner := range t.tools {
		vers, err := inner.ListVersions(ctx)
		if err != nil {
			// One platform repo may be empty or temporarily unavailable; keep going.
			continue
		}
		for _, v := range vers {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			collected = append(collected, v)
		}
	}
	return install.SortVersionsDesc(collected)
}

func (t *heliumBrowserTool) Install(ctx context.Context, version string, destDir string) error {
	return install.InstallSelectedArtifact(ctx, version, destDir, "helium", "registry:helium-browser", strings.TrimSpace, t.ListVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *heliumBrowserTool) Pin() tool.Pin {
	var pin tool.Pin
	pin.Versioning = "semver"

	return pin
}

func (t *heliumBrowserTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	v, err := install.ResolveToolVersion(ctx, version, strings.TrimSpace, t.ListVersions)
	if err != nil {
		return nil, err
	}

	var all []tool.Artifact
	for _, inner := range t.tools {
		at, ok := inner.(tool.ArtifactTool)
		if !ok {
			continue
		}
		arts, err := at.ListArtifacts(ctx, v)
		if err != nil {
			// The version tag may not exist in this OS-specific repo.
			continue
		}
		all = append(all, arts...)
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
