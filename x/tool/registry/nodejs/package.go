package nodejs

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

type nodejsTool struct{}

func newNodejs() (tool.Tool, error) {
	return &nodejsTool{}, nil
}

func (t *nodejsTool) ListVersions(ctx context.Context) ([]string, error) {
	return t.listVersions(ctx)
}

func (t *nodejsTool) Install(ctx context.Context, version string, destDir string) error {
	return install.InstallFirstArtifact(ctx, version, destDir, normalizeNodejsVersion, t.listVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *nodejsTool) Pin() tool.Pin {
	var pin tool.Pin
	// No standard renovate datasource for direct nodejs.org; shasums give us
	// verification at install time via fetchurl.

	return pin
}

func (t *nodejsTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	v, err := install.ResolveToolVersion(ctx, version, normalizeNodejsVersion, t.listVersions)
	if err != nil {
		return nil, err
	}

	osPart, archPart, ext := t.nodePlatformAndExt()
	filename := fmt.Sprintf("node-%s-%s-%s%s", v, osPart, archPart, ext)
	url := fmt.Sprintf("https://nodejs.org/dist/%s/%s", v, filename)

	// Fetch SHASUMS256.txt so we can attach hash and use fetchurl tool.
	sums, err := t.fetchShasums(ctx, v)
	if err != nil {
		return nil, err
	}
	hash := ""
	if h, ok := sums[filename]; ok && h != "" {
		hash = "sha256:" + h
	}

	return []tool.Artifact{{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		URL:  url,
		Hash: hash,
	}}, nil
}

func (t *nodejsTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	return install.DefaultInstallArtifact(ctx, artifact, destDir)
}

func (t *nodejsTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	return tool.EnsureBinary(destDir, cmdName, "Node.js", func() error {
		return t.Install(ctx, version, destDir)
	}, "node")
}

// --- helpers ---

func (t *nodejsTool) listVersions(ctx context.Context) ([]string, error) {
	var infos []struct {
		Version string `json:"version"`
	}
	if err := install.GetJSON(ctx, "https://nodejs.org/dist/index.json", &infos); err != nil {
		return nil, err
	}
	out := make([]string, len(infos))
	for i, v := range infos {
		out[i] = v.Version
	}
	return out, nil
}

func (t *nodejsTool) fetchShasums(ctx context.Context, ver string) (map[string]string, error) {
	ver = normalizeNodejsVersion(ver)
	u := fmt.Sprintf("https://nodejs.org/dist/%s/SHASUMS256.txt", ver)
	b, err := install.GetBytes(ctx, u)
	if err != nil {
		return nil, err
	}
	return install.ParseGNUHashFile(b), nil
}

func (t *nodejsTool) nodePlatformAndExt() (osPart, archPart, ext string) {
	osPart = runtime.GOOS
	archPart = runtime.GOARCH
	ext = ".tar.gz"

	switch osPart {
	case "darwin":
		osPart = "darwin"
	case "linux":
		osPart = "linux"
	case "windows":
		osPart = "win"
		ext = ".zip"
	}

	switch archPart {
	case "amd64":
		archPart = "x64"
	case "arm64":
		archPart = "arm64"
	case "386":
		archPart = "x86"
	}

	return osPart, archPart, ext
}

func normalizeNodejsVersion(version string) string {
	v := strings.TrimSpace(version)
	if v == "" || v == "latest" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

func (t *nodejsTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("node"))
}
