package pkg

import (
	"context"
	"fmt"
	stdpath "path"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

var ErrNoSuitableArtifactFoundFor = install.ErrNoSuitableArtifactFoundFor

// microsandboxTool installs the release bundle, not the standalone msb asset.
// The standalone binary has no libkrunfw, so it cannot boot a sandbox.
type microsandboxTool struct {
	inner tool.Tool
}

func newMicrosandbox() (tool.Tool, error) {
	inner, err := github.NewTool("superradcompany/microsandbox")
	if err != nil {
		return nil, err
	}
	return &microsandboxTool{inner: inner}, nil
}

func (t *microsandboxTool) ListVersions(ctx context.Context) ([]string, error) {
	versions, err := t.inner.ListVersions(ctx)
	if err != nil {
		return nil, err
	}
	return install.SortVersionsDesc(install.KeepVersions(versions, microsandboxVersionFromTag))
}

func (t *microsandboxTool) Install(ctx context.Context, version, destination string) error {
	resolved, err := install.ResolveToolVersion(ctx, version, normalizeMicrosandboxVersion, t.ListVersions)
	if err != nil {
		return err
	}
	artifacts, err := t.ListArtifacts(ctx, resolved)
	if err != nil {
		return err
	}
	artifact := selectMicrosandboxArtifact(artifacts, runtime.GOOS, runtime.GOARCH)
	if artifact == nil {
		return fmt.Errorf("%w %s/%s for registry:microsandbox@%s", ErrNoSuitableArtifactFoundFor, runtime.GOOS, runtime.GOARCH, resolved)
	}
	return t.InstallArtifact(ctx, *artifact, destination)
}

func (t *microsandboxTool) Pin() tool.Pin {
	return tool.InheritSemver(t.inner)
}

func (t *microsandboxTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	resolved, err := install.ResolveToolVersion(ctx, version, normalizeMicrosandboxVersion, t.ListVersions)
	if err != nil {
		return nil, err
	}
	artifactTool, ok := t.inner.(tool.ArtifactTool)
	if !ok {
		return nil, registry.ErrNoArtifactTool
	}
	artifacts, err := artifactTool.ListArtifacts(ctx, "v"+resolved)
	if err != nil {
		return nil, err
	}
	// The store selects from this list, so hide the standalone msb asset that
	// lacks libkrunfw. Only the release bundle can boot a sandbox.
	out := make([]tool.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		if isMicrosandboxBundle(artifact.URL) {
			out = append(out, artifact)
		}
	}
	return out, nil
}

func (t *microsandboxTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destination string) error {
	artifactTool, ok := t.inner.(tool.ArtifactTool)
	if !ok {
		return registry.ErrNoArtifactTool
	}
	return artifactTool.InstallArtifact(ctx, artifact, destination)
}

func (t *microsandboxTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("msb"))
}

// microsandboxVersionFromTag returns the semver part of a release tag.
// Prereleases are skipped; the CLI and the guest runtime ship together.
func microsandboxVersionFromTag(tag string) (string, bool) {
	version := strings.TrimPrefix(strings.TrimSpace(tag), "v")
	if version == "" || version == tag || strings.Contains(version, "-") {
		return "", false
	}
	return version, true
}

func normalizeMicrosandboxVersion(version string) string {
	return install.NormalizeV(version)
}

func isMicrosandboxBundle(url string) bool {
	base := strings.ToLower(stdpath.Base(url))
	if !strings.HasPrefix(base, "microsandbox-") {
		return false
	}
	return strings.HasSuffix(base, ".tar.gz") || strings.HasSuffix(base, ".zip")
}

// selectMicrosandboxArtifact picks the platform release bundle that contains
// msb and libkrunfw. Standalone assets such as msb-linux-x86_64 are rejected.
func selectMicrosandboxArtifact(artifacts []tool.Artifact, goos, goarch string) *tool.Artifact {
	var best *tool.Artifact
	bestScore := -1
	for i := range artifacts {
		artifact := &artifacts[i]
		if artifact.OS != goos || artifact.Arch != goarch {
			continue
		}
		if !isMicrosandboxBundle(artifact.URL) {
			continue
		}
		base := strings.ToLower(stdpath.Base(artifact.URL))
		score := 100
		switch {
		case strings.HasSuffix(base, ".tar.gz"):
			score += 20
		case strings.HasSuffix(base, ".zip"):
			score += 10
		}
		if score > bestScore {
			bestScore = score
			best = artifact
		}
	}
	return best
}
