// Package registry is the short-name backend.
//
// A bare spec such as uv@0.5.0 uses backend id registry. Names are registered
// with RegisterTool. RegisterGitHub wraps a GitHub Releases repository with
// v-prefix tag fallback and install checks.
package registry

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
)

var (
	// ErrEmptyToolName is returned when a curated name is blank.
	ErrEmptyToolName = errors.New("curated tool name cannot be empty")
	// ErrNoArtifactTool is returned when a curated tool's inner backend cannot list artifacts.
	ErrNoArtifactTool = errors.New("inner tool does not implement ArtifactTool")
)

var namedTools = map[string]func() (tool.Tool, error){}

// RegisterTool registers a curated short name such as "uv".
// A second registration of the same name panics.
func RegisterTool(name string, construct func() (tool.Tool, error)) {
	if _, ok := namedTools[name]; ok {
		panic(fmt.Sprintf("registry: tool %s is being defined twice", name))
	}
	namedTools[name] = construct
}

type curatedGitHub struct {
	inner      tool.Tool
	binaryHint string
	checks     []tool.Check
}

func newCuratedGitHub(repo, binaryHint string, toolChecks ...tool.Check) (tool.Tool, error) {
	inner, err := github.NewTool(repo, binaryHint)
	if err != nil {
		return nil, err
	}
	return &curatedGitHub{inner: inner, binaryHint: binaryHint, checks: tool.Checks(toolChecks...)}, nil
}

func (installed *curatedGitHub) ListVersions(ctx context.Context) ([]string, error) {
	versions, err := installed.inner.ListVersions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(versions))
	for _, version := range versions {
		out = append(out, strings.TrimPrefix(strings.TrimSpace(version), "v"))
	}
	return out, nil
}

func (installed *curatedGitHub) Install(ctx context.Context, version, destination string) error {
	version = strings.TrimSpace(version)
	try := func(candidate string) error {
		if artifactTool, ok := installed.inner.(tool.ArtifactTool); ok && installed.binaryHint != "" {
			artifacts, err := artifactTool.ListArtifacts(ctx, candidate)
			if err != nil {
				return err
			}
			if chosen := tool.SelectArtifact(artifacts, runtime.GOOS, runtime.GOARCH, installed.binaryHint); chosen != nil {
				return artifactTool.InstallArtifact(ctx, *chosen, destination)
			}
			if len(artifacts) > 0 {
				return fmt.Errorf("no suitable artifact found for %s/%s for registry@%s: %w", runtime.GOOS, runtime.GOARCH, candidate, github.ErrNoArtifact)
			}
		}
		return installed.inner.Install(ctx, candidate, destination)
	}
	if version == "" || version == "latest" {
		return try(version)
	}
	if !strings.HasPrefix(version, "v") {
		if err := try("v" + version); err == nil {
			return nil
		} else if !errors.Is(err, github.ErrAPIError) {
			return err
		}
	}
	return try(version)
}

func (installed *curatedGitHub) Pin() tool.Pin {
	if pinner, ok := installed.inner.(tool.Pinner); ok {
		return pinner.Pin()
	}
	return tool.Pin{}
}

func (installed *curatedGitHub) InstallChecks() []tool.Check {
	return tool.Checks(installed.checks...)
}

func (installed *curatedGitHub) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	artifactTool, ok := installed.inner.(tool.ArtifactTool)
	if !ok {
		return nil, ErrNoArtifactTool
	}
	return artifactTool.ListArtifacts(ctx, version)
}

func (installed *curatedGitHub) InstallArtifact(ctx context.Context, artifact tool.Artifact, destination string) error {
	artifactTool, ok := installed.inner.(tool.ArtifactTool)
	if !ok {
		return ErrNoArtifactTool
	}
	return artifactTool.InstallArtifact(ctx, artifact, destination)
}

func (installed *curatedGitHub) Fix(ctx context.Context, destination string) error {
	fixer, ok := installed.inner.(tool.Fixer)
	if !ok {
		return nil
	}
	return fixer.Fix(ctx, destination)
}

// RegisterGitHub registers a github-backed short name.
// An empty check list defaults to Binary(name).
// Artifact selection prefers the binary named by the first Binary check.
func RegisterGitHub(name, repo string, toolChecks ...tool.Check) {
	if len(toolChecks) == 0 {
		toolChecks = tool.Checks(tool.Binary(name))
	}
	checksCopy := tool.Checks(toolChecks...)
	hint := binaryHintFromChecks(name, checksCopy)
	RegisterTool(name, func() (tool.Tool, error) {
		return newCuratedGitHub(repo, hint, checksCopy...)
	})
}

func binaryHintFromChecks(fallback string, list []tool.Check) string {
	for _, check := range list {
		if name := check.Name(); strings.HasPrefix(name, "binary:") {
			if hint := strings.TrimPrefix(name, "binary:"); hint != "" {
				return hint
			}
		}
	}
	return fallback
}

type catalog struct{}

func (catalog) Name() string { return "Tool Catalog (curated short names)" }

func (catalog) Tool(ref string) (tool.Tool, error) {
	name := strings.TrimSpace(ref)
	if name == "" {
		return nil, ErrEmptyToolName
	}
	construct, ok := namedTools[name]
	if !ok {
		return nil, fmt.Errorf("unknown named tool %q (the catalog only knows curated short names; bare names default to the catalog; use explicit 'mise:xxx' or 'github:owner/repo' for other tools)", name)
	}
	return construct()
}

// NewTool constructs the curated tool registered as ref.
func NewTool(ref string) (tool.Tool, error) {
	return catalog{}.Tool(ref)
}

// ListTools returns the sorted curated short names.
func ListTools() []string {
	names := make([]string, 0, len(namedTools))
	for name := range namedTools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
