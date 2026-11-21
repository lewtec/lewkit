package applications

import (
	"context"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry"
	"strings"
)

func init() {
	registry.RegisterTool("tirith", newTirith)
}

type tirithTool struct {
	inner tool.Tool
}

func newTirith() (tool.Tool, error) {
	inner, err := github.NewTool("sheeki03/tirith")
	if err != nil {
		return nil, err
	}
	return tirithTool{inner: inner}, nil
}

func (t tirithTool) ListVersions(ctx context.Context) ([]string, error) {
	versions, err := t.inner.ListVersions(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(versions))
	for _, version := range versions {
		if isTirithProgramVersion(version) {
			out = append(out, version)
		}
	}
	return out, nil
}

func (t tirithTool) Install(ctx context.Context, version string, destDir string) error {
	return t.inner.Install(ctx, version, destDir)
}

func (t tirithTool) Pin() tool.Pin {
	var pin tool.Pin
	if pinner, ok := t.inner.(tool.Pinner); ok {
		pin = pinner.Pin()
	}
	pin.Versioning = "semver"

	return pin
}

func isTirithProgramVersion(version string) bool {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func (t tirithTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("tirith"))
}
