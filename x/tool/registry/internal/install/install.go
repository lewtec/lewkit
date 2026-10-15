// Package install holds the download and version helpers shared by curated tools.
package install

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/httpclient"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/tool"
)

var (
	// ErrNoVersions is returned when a catalog tool lists no versions.
	ErrNoVersions = tool.ErrNoVersionsFound
	// ErrNoPlatformArtifact is returned when the current OS and arch have no artifact.
	ErrNoPlatformArtifact         = errs.Wrap(errs.ErrNotFound, "no artifact for current platform")
	ErrNoSuitableArtifactFoundFor = errs.Wrap(errs.ErrNotFound, "no suitable artifact found for")
)

// Host returns goos and goarch, using the process pair when either value is empty.
func Host(goos, goarch string) (string, string) {
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	return goos, goarch
}

// CollectVersions reads one version from each item, then applies UniqueVersions.
func CollectVersions[T any](items []T, version func(T) string) ([]string, error) {
	versions := make([]string, len(items))
	for i, item := range items {
		versions[i] = version(item)
	}
	return UniqueVersions(versions)
}

// UniqueVersions drops blanks and repeats, then orders the rest newest-first.
// Empty input yields ErrNoVersions.
func UniqueVersions(versions []string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(versions))
	for _, version := range versions {
		if version == "" || seen[version] {
			continue
		}
		seen[version] = true
		out = append(out, version)
	}
	return SortVersionsDesc(out)
}

// SortVersionsDesc returns versions ordered newest-first by semver so [0] is
// latest. Empty input yields ErrNoVersions.
func SortVersionsDesc(versions []string) ([]string, error) {
	if len(versions) == 0 {
		return nil, ErrNoVersions
	}
	out := append([]string(nil), versions...)
	sort.Slice(out, func(i, j int) bool {
		return tool.CompareVersions(out[i], out[j]) > 0
	})
	return out, nil
}

// NormalizeVersion trims space and the given prefixes (each once, in order).
// Empty and "latest" are returned unchanged after that strip.
func NormalizeVersion(version string, prefixes ...string) string {
	v := strings.TrimSpace(version)
	for _, p := range prefixes {
		v = strings.TrimPrefix(v, p)
	}
	if v == "" || v == "latest" {
		return v
	}
	return v
}

// NormalizeVPrefixed strips a leading v/V (Flutter, CMake, …).
func NormalizeVPrefixed(version string) string {
	return NormalizeVersion(version, "v", "V")
}

// NormalizeV strips a leading v only (esbuild, terraform, …).
func NormalizeV(version string) string {
	return NormalizeVersion(version, "v")
}

// DefaultInstallArtifact is the usual InstallArtifact body for registry tools
// that download via providerinstall with default DownloadOptions.
func DefaultInstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	return tool.InstallArtifact(ctx, artifact, destDir, tool.DownloadOptions{})
}

// EnsureToolBinary is the usual EnsureBinary body: install then locate cmdName.
func EnsureToolBinary(ctx context.Context, version, cmdName, destDir, label string, install func(context.Context, string, string) error) (string, error) {
	return tool.EnsureBinary(destDir, cmdName, label, func() error {
		return install(ctx, version, destDir)
	})
}

func ResolveToolVersion(ctx context.Context, version string, normalize func(string) string, listVersions func(context.Context) ([]string, error)) (string, error) {
	v := normalize(version)
	if v != "" && v != "latest" {
		return v, nil
	}
	vers, err := listVersions(ctx)
	if err != nil {
		return "", err
	}
	if len(vers) == 0 {
		return "", ErrNoVersions
	}
	return normalize(vers[0]), nil
}

func InstallFirstArtifact(
	ctx context.Context,
	version, destDir string,
	normalize func(string) string,
	listVersions func(context.Context) ([]string, error),
	listArtifacts func(context.Context, string) ([]tool.Artifact, error),
	installArtifact func(context.Context, tool.Artifact, string) error,
) error {
	v, err := ResolveToolVersion(ctx, version, normalize, listVersions)
	if err != nil {
		return err
	}
	arts, err := listArtifacts(ctx, v)
	if err != nil {
		return err
	}
	if len(arts) == 0 {
		return ErrNoPlatformArtifact
	}
	return installArtifact(ctx, arts[0], destDir)
}

func InstallSelectedArtifact(
	ctx context.Context,
	version, destDir, binaryHint, toolRef string,
	normalize func(string) string,
	listVersions func(context.Context) ([]string, error),
	listArtifacts func(context.Context, string) ([]tool.Artifact, error),
	installArtifact func(context.Context, tool.Artifact, string) error,
) error {
	v, err := ResolveToolVersion(ctx, version, normalize, listVersions)
	if err != nil {
		return err
	}
	arts, err := listArtifacts(ctx, v)
	if err != nil {
		return err
	}
	if len(arts) == 0 {
		return ErrNoPlatformArtifact
	}
	artifact := tool.SelectArtifact(arts, runtime.GOOS, runtime.GOARCH, binaryHint)
	if artifact == nil {
		return fmt.Errorf("%w %s/%s for %s@%s", ErrNoSuitableArtifactFoundFor, runtime.GOOS, runtime.GOARCH, toolRef, v)
	}
	return installArtifact(ctx, *artifact, destDir)
}

type unexpectedHTTPStatusError struct {
	url    string
	status string
}

func (e unexpectedHTTPStatusError) Error() string {
	return "GET " + e.url + ": " + e.status
}

func httpGET(ctx context.Context, u string, configure ...func(*http.Request)) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	for _, fn := range configure {
		if fn != nil {
			fn(req)
		}
	}
	httpDriver, err := driver.Get[httpclient.Driver](ctx)
	if err != nil {
		return nil, err
	}
	resp, err := httpDriver.Client().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, unexpectedHTTPStatusError{url: u, status: resp.Status}
	}
	return resp, nil
}

// GetBytesOr returns hook's bytes when hook is set, and otherwise GET u.
func GetBytesOr(ctx context.Context, u string, hook func(context.Context, string) ([]byte, error)) ([]byte, error) {
	if hook != nil {
		return hook(ctx, u)
	}
	return GetBytes(ctx, u)
}

func GetBytes(ctx context.Context, u string, configure ...func(*http.Request)) ([]byte, error) {
	resp, err := httpGET(ctx, u, configure...)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func GetJSON(ctx context.Context, u string, dest any) error {
	resp, err := httpGET(ctx, u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(dest)
}

// ParseGNUHashFile maps filename → digest from GNU coreutils hash listings
// (SHA-256.txt, SHASUMS256.txt): one "digest  filename" pair per line.
func ParseGNUHashFile(b []byte) map[string]string {
	m := map[string]string{}
	for line := range strings.SplitSeq(string(b), "\n") {
		fs := strings.Fields(line)
		if len(fs) >= 2 {
			m[fs[1]] = fs[0]
		}
	}
	return m
}
