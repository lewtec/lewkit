package esbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

// esbuild is installed from the @esbuild/* platform packages on the npm
// registry (same layout mise uses for http:esbuild). No Node runtime is
// required: each package is a tarball with bin/esbuild (or esbuild.exe).

const esbuildRegistryBase = "https://registry.npmjs.org"

type esbuildTool struct{}

func newEsbuild() (tool.Tool, error) {
	return &esbuildTool{}, nil
}

func (t *esbuildTool) ListVersions(ctx context.Context) ([]string, error) {
	return t.listVersions(ctx)
}

func (t *esbuildTool) Install(ctx context.Context, version string, destDir string) error {
	return install.InstallFirstArtifact(ctx, version, destDir, normalizeEsbuildVersion, t.listVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *esbuildTool) Pin() tool.Pin {
	var pin tool.Pin
	pin.Name = "esbuild"
	pin.Datasource = "npm"
	pin.Versioning = "semver"

	return pin
}

func (t *esbuildTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	v, err := install.ResolveToolVersion(ctx, version, normalizeEsbuildVersion, t.listVersions)
	if err != nil {
		return nil, err
	}

	plat, ok := esbuildPlatform(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", install.ErrNoPlatformArtifact, runtime.GOOS, runtime.GOARCH)
	}

	url := esbuildArtifactURL(plat, v)
	hash := t.fetchTarballHash(ctx, plat, v)

	return []tool.Artifact{{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		URL:  url,
		Hash: hash,
	}}, nil
}

func (t *esbuildTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	return install.DefaultInstallArtifact(ctx, artifact, destDir)
}

func (t *esbuildTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	return tool.EnsureBinaryNamed(destDir, cmdName, "esbuild", "esbuild", func() error {
		return t.Install(ctx, version, destDir)
	})
}

func (t *esbuildTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("esbuild"))
}

func (t *esbuildTool) listVersions(ctx context.Context) ([]string, error) {
	body, err := esbuildGET(ctx, esbuildRegistryBase+"/esbuild")
	if err != nil {
		return nil, err
	}

	var packument struct {
		Versions map[string]json.RawMessage `json:"versions"`
	}
	if err := json.Unmarshal(body, &packument); err != nil {
		return nil, err
	}
	if len(packument.Versions) == 0 {
		return nil, install.ErrNoVersions
	}

	out := make([]string, 0, len(packument.Versions))
	for ver := range packument.Versions {
		ver = strings.TrimSpace(ver)
		// Stable releases only (skip 0.19.0-beta, etc.).
		if ver == "" || strings.Contains(ver, "-") {
			continue
		}
		out = append(out, ver)
	}
	return install.SortVersionsDesc(out)
}

// fetchTarballHash returns "sha1:<shasum>" from the platform package document.
// Failures are non-fatal: install still works without a hash (direct download).
func (t *esbuildTool) fetchTarballHash(ctx context.Context, plat, version string) string {
	u := fmt.Sprintf("%s/@esbuild/%s/%s", esbuildRegistryBase, plat, version)
	body, err := esbuildGET(ctx, u)
	if err != nil {
		return ""
	}
	var doc struct {
		Dist struct {
			Shasum string `json:"shasum"`
		} `json:"dist"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return ""
	}
	if doc.Dist.Shasum == "" {
		return ""
	}
	return "sha1:" + doc.Dist.Shasum
}

func esbuildGET(ctx context.Context, u string) ([]byte, error) {
	return install.GetBytes(ctx, u, func(req *http.Request) {
		// Prefer install-v1 packument (smaller) when the registry supports it.
		req.Header.Set("Accept", "application/vnd.npm.install-v1+json; q=1.0, application/json; q=0.8")
	})
}

func esbuildArtifactURL(plat, version string) string {
	// https://registry.npmjs.org/@esbuild/linux-x64/-/linux-x64-0.28.1.tgz
	return fmt.Sprintf("%s/@esbuild/%s/-/%s-%s.tgz", esbuildRegistryBase, plat, plat, version)
}

func normalizeEsbuildVersion(version string) string {
	return install.NormalizeV(version)
}

// esbuildPlatform maps GOOS/GOARCH to the @esbuild/<platform> package suffix
// (npm/esbuild naming: darwin not macos, win32 not windows, x64 not amd64).
func esbuildPlatform(goos, goarch string) (plat string, ok bool) {
	var osPart string
	switch goos {
	case "linux":
		osPart = "linux"
	case "darwin":
		osPart = "darwin"
	case "windows":
		osPart = "win32"
	case "freebsd":
		osPart = "freebsd"
	case "netbsd":
		osPart = "netbsd"
	case "openbsd":
		osPart = "openbsd"
	default:
		return "", false
	}

	var archPart string
	switch goarch {
	case "amd64":
		archPart = "x64"
	case "arm64":
		archPart = "arm64"
	case "arm":
		archPart = "arm"
	case "386":
		archPart = "ia32"
	case "ppc64":
		archPart = "ppc64"
	case "riscv64":
		archPart = "riscv64"
	case "s390x":
		archPart = "s390x"
	case "loong64":
		archPart = "loong64"
	case "mips64le":
		archPart = "mips64el"
	default:
		return "", false
	}

	return osPart + "-" + archPart, true
}
