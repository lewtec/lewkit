package applications

import (
	"context"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry"
	"os"
	"runtime"
	"strings"
)

const bendOrigin = "https://bend-lang.com"

var ErrInvalidBendVersion = errors.New("invalid bend version")

func init() {
	registry.RegisterTool("bend", newBend)
}

// bendTool installs Bend 2 from bend-lang.com tarballs. The official site
// only ships curl|sh, which also phones home. bun is not copied into the
// dest dir; the launcher re-enters workspaced (tool with bun) at runtime.
type bendTool struct {
	origin   string
	fetchURL func(context.Context, string) ([]byte, error)
}

type bendRelease struct {
	Ver    string `json:"ver"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

func newBend() (tool.Tool, error) {
	return &bendTool{origin: bendOrigin}, nil
}

func (t *bendTool) baseURL() string {
	if o := strings.TrimRight(strings.TrimSpace(t.origin), "/"); o != "" {
		return o
	}
	return bendOrigin
}

func (t *bendTool) ListVersions(ctx context.Context) ([]string, error) {
	rel, err := t.fetchLatest(ctx)
	if err != nil {
		return nil, err
	}
	return []string{rel.Ver}, nil
}

func (t *bendTool) Install(ctx context.Context, version string, destDir string) error {
	return installFirstArtifact(ctx, version, destDir, normalizeV, t.ListVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *bendTool) Pin() tool.Pin {
	var pin tool.Pin
	pin.Versioning = "semver"

	return pin
}

func (t *bendTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	if runtime.GOOS == "windows" {
		return nil, ErrNoPlatformArtifact
	}

	v := normalizeV(version)
	rel, relErr := t.fetchLatest(ctx)
	if v == "" || v == "latest" {
		if relErr != nil {
			return nil, relErr
		}
		v = rel.Ver
	}
	if !validBendVersion(v) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidBendVersion, version)
	}

	u := fmt.Sprintf("%s/dl/%s.tar.gz", t.baseURL(), v)
	hash := ""
	if relErr == nil && rel.Ver == v {
		if rel.URL != "" {
			u = rel.URL
		}
		if rel.SHA256 != "" {
			hash = "sha256:" + rel.SHA256
		}
	}

	return []tool.Artifact{{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		URL:  u,
		Hash: hash,
	}}, nil
}

func (t *bendTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	if err := defaultInstallArtifact(ctx, artifact, destDir); err != nil {
		return err
	}
	return writeBendLauncher(destDir)
}

func (t *bendTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	return ensureToolBinary(ctx, version, cmdName, destDir, "Bend", t.Install)
}

func (t *bendTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("bend"))
}

func (t *bendTool) fetchLatest(ctx context.Context) (bendRelease, error) {
	body, err := t.fetch(ctx, t.baseURL()+"/dl/latest.json")
	if err != nil {
		return bendRelease{}, err
	}
	var rel bendRelease
	if err := jsonv2.Unmarshal(body, &rel); err != nil {
		return bendRelease{}, fmt.Errorf("decode bend latest.json: %w", err)
	}
	if !validBendVersion(rel.Ver) {
		return bendRelease{}, fmt.Errorf("%w in latest.json: %q", ErrInvalidBendVersion, rel.Ver)
	}
	return rel, nil
}

func (t *bendTool) fetch(ctx context.Context, u string) ([]byte, error) {
	if t.fetchURL != nil {
		return t.fetchURL(ctx, u)
	}
	return getBytes(ctx, u)
}

func validBendVersion(v string) bool {
	if v == "" || v[0] < '0' || v[0] > '9' || v[len(v)-1] < '0' || v[len(v)-1] > '9' {
		return false
	}
	prevDot := false
	for _, r := range v {
		switch {
		case r >= '0' && r <= '9':
			prevDot = false
		case r == '.':
			if prevDot {
				return false
			}
			prevDot = true
		default:
			return false
		}
	}
	return true
}

const bendLauncherScript = `#!/bin/sh
set -eu
bindir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
root=$(CDPATH= cd -- "$bindir/.." && pwd)
main=""
for cand in "$root/bend2/main.ts" "$root/main.ts"; do
	if [ -f "$cand" ]; then
		main=$cand
		break
	fi
done
if [ -z "$main" ]; then
	echo "bend: missing bend2/main.ts under $root" >&2
	exit 1
fi
ws=$(command -v workspaced) || {
	echo "bend: workspaced not found on PATH" >&2
	exit 1
}
export BEND_NO_TELEMETRY=1
exec -a bun "$ws" tool with bun -- bun "$main" "$@"
`

func writeBendLauncher(destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	root, err := path.Open(destDir)
	if err != nil {
		return err
	}
	defer root.Close()
	launcher := path.New("bin", "bend")
	if err := launcher.Parent().MkdirAll(root, 0o755); err != nil {
		return err
	}
	return launcher.WriteFile(root, []byte(bendLauncherScript), 0o755)
}
