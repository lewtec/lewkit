package pkg

import (
	"context"
	"fmt"
	"path"
	"regexp"

	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

const blenderReleaseBaseURL = "https://download.blender.org/release/"

var (
	hrefPattern          = regexp.MustCompile(`href="([^"]+)"`)
	blenderSeriesPattern = regexp.MustCompile(`^Blender[0-9]+\.[0-9]+$`)
)

type blenderTool struct {
	baseURL  string
	fetchURL func(context.Context, string) ([]byte, error)
	goos     string
	goarch   string
}

type blenderBuild struct {
	Version string
	Name    string
	URL     string
}

func newBlender() (tool.Tool, error) {
	return &blenderTool{}, nil
}

func (t *blenderTool) ListVersions(ctx context.Context) ([]string, error) {
	goos, goarch := install.Host(t.goos, t.goarch)
	if !blenderPlatformSupported(goos, goarch) {
		return nil, fmt.Errorf("%w: %s/%s", ErrNoPlatformArtifact, goos, goarch)
	}
	builds, err := t.allBuilds(ctx, goos, goarch)
	if err != nil {
		return nil, err
	}
	return blenderVersions(builds)
}

func (t *blenderTool) Install(ctx context.Context, version, destDir string) error {
	return install.InstallFirstArtifact(ctx, version, destDir, normalizeBlenderVersion, t.ListVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *blenderTool) Pin() tool.Pin {
	return tool.SemverPin()
}

func (t *blenderTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	goos, goarch := install.Host(t.goos, t.goarch)
	if !blenderPlatformSupported(goos, goarch) {
		return nil, fmt.Errorf("%w: %s/%s", ErrNoPlatformArtifact, goos, goarch)
	}
	resolved, err := install.ResolveToolVersion(ctx, version, normalizeBlenderVersion, t.ListVersions)
	if err != nil {
		return nil, err
	}
	series, ok := blenderSeries(resolved)
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrNoPlatformArtifact, goos, goarch)
	}
	names, err := t.seriesFiles(ctx, series)
	if err != nil {
		return nil, err
	}
	for _, build := range blenderBuilds(t.seriesURL(series), names, goos, goarch) {
		if build.Version != resolved {
			continue
		}
		return []tool.Artifact{{
			OS:   goos,
			Arch: goarch,
			URL:  build.URL,
			Hash: t.archiveHash(ctx, series, resolved, build.Name),
		}}, nil
	}
	return nil, fmt.Errorf("%w: %s/%s", ErrNoPlatformArtifact, goos, goarch)
}

func (t *blenderTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	return install.DefaultInstallArtifact(ctx, artifact, destDir)
}

func (t *blenderTool) EnsureBinary(ctx context.Context, version, cmdName, destDir string) (string, error) {
	return install.EnsureToolBinary(ctx, version, cmdName, destDir, "Blender", t.Install)
}

func (t *blenderTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("blender"))
}

func (t *blenderTool) releaseBase() string {
	base := strings.TrimSpace(t.baseURL)
	if base == "" {
		base = blenderReleaseBaseURL
	}
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base
}

func (t *blenderTool) seriesURL(series string) string {
	return t.releaseBase() + series + "/"
}

func (t *blenderTool) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	if t.fetchURL != nil {
		return t.fetchURL(ctx, rawURL)
	}
	return install.GetBytes(ctx, rawURL)
}

func (t *blenderTool) allBuilds(ctx context.Context, goos, goarch string) ([]blenderBuild, error) {
	body, err := t.fetch(ctx, t.releaseBase())
	if err != nil {
		return nil, err
	}
	seriesNames := blenderSeriesNames(body)
	if len(seriesNames) == 0 {
		return nil, ErrNoVersions
	}
	listings, err := t.fetchSeries(ctx, seriesNames)
	if err != nil {
		return nil, err
	}
	var builds []blenderBuild
	for _, series := range seriesNames {
		builds = append(builds, blenderBuilds(t.seriesURL(series), listings[series], goos, goarch)...)
	}
	return builds, nil
}

type seriesListing struct {
	series string
	files  []string
	err    error
}

func (t *blenderTool) fetchSeries(ctx context.Context, seriesNames []string) (map[string][]string, error) {
	results := make(chan seriesListing, len(seriesNames))
	var wait sync.WaitGroup
	limit := make(chan struct{}, 8)
	for _, series := range seriesNames {
		wait.Add(1)
		go func(series string) {
			defer wait.Done()
			select {
			case <-ctx.Done():
				results <- seriesListing{series: series, err: context.Cause(ctx)}
				return
			case limit <- struct{}{}:
			}
			defer func() { <-limit }()
			files, err := t.seriesFiles(ctx, series)
			results <- seriesListing{series: series, files: files, err: err}
		}(series)
	}
	go func() {
		wait.Wait()
		close(results)
	}()

	listings := make(map[string][]string, len(seriesNames))
	var first error
	for result := range results {
		if result.err != nil {
			if first == nil {
				first = result.err
			}
			continue
		}
		listings[result.series] = result.files
	}
	if first != nil {
		return nil, first
	}
	return listings, nil
}

func (t *blenderTool) seriesFiles(ctx context.Context, series string) ([]string, error) {
	body, err := t.fetch(ctx, t.seriesURL(series))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", series, err)
	}
	return indexFileNames(body), nil
}

func (t *blenderTool) archiveHash(ctx context.Context, series, version, fileName string) string {
	body, err := t.fetch(ctx, t.seriesURL(series)+"blender-"+version+".sha256")
	if err != nil {
		return ""
	}
	sum := install.ParseGNUHashFile(body)[fileName]
	if sum == "" {
		return ""
	}
	return "sha256:" + sum
}

func normalizeBlenderVersion(version string) string {
	return install.NormalizeVPrefixed(version)
}

func blenderSeries(version string) (string, bool) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return "Blender" + parts[0] + "." + parts[1], true
}

func blenderPlatformSupported(goos, goarch string) bool {
	switch goos {
	case "linux", "darwin", "windows":
	default:
		return false
	}
	switch goarch {
	case "amd64", "arm64":
		return true
	default:
		return false
	}
}

func blenderVersions(builds []blenderBuild) ([]string, error) {
	return install.CollectVersions(builds, func(build blenderBuild) string { return build.Version })
}

func blenderBuilds(seriesURL string, names []string, goos, goarch string) []blenderBuild {
	best := map[string]blenderBuild{}
	bestRank := map[string]int{}
	for _, name := range names {
		version, osName, arch, rank, ok := parseBlenderName(name)
		if !ok || osName != goos || arch != goarch {
			continue
		}
		if previous, exists := bestRank[version]; exists && previous >= rank {
			continue
		}
		bestRank[version] = rank
		fileName := path.Base(name)
		best[version] = blenderBuild{
			Version: version,
			Name:    fileName,
			URL:     seriesURL + fileName,
		}
	}
	out := make([]blenderBuild, 0, len(best))
	for _, build := range best {
		out = append(out, build)
	}
	return out
}

func parseBlenderName(name string) (version, goos, goarch string, rank int, ok bool) {
	base := strings.ToLower(path.Base(name))
	rest, found := strings.CutPrefix(base, "blender-")
	if !found {
		return "", "", "", 0, false
	}
	version, rest, ok = splitBlenderVersion(rest)
	if !ok {
		return "", "", "", 0, false
	}
	rank = blenderArchiveRank(base)
	if rank == 0 {
		return "", "", "", 0, false
	}
	goos, goarch, ok = blenderOSArch(rest)
	if !ok {
		return "", "", "", 0, false
	}
	return version, goos, goarch, rank, true
}

func splitBlenderVersion(rest string) (string, string, bool) {
	end := 0
	dots := 0
	for end < len(rest) && ((rest[end] >= '0' && rest[end] <= '9') || rest[end] == '.') {
		if rest[end] == '.' {
			dots++
		}
		end++
	}
	if dots != 2 || end == 0 || rest[end-1] == '.' {
		return "", "", false
	}
	if end < len(rest) && rest[end] != '-' && rest[end] != '.' {
		return "", "", false
	}
	return rest[:end], rest[end:], true
}

func blenderOSArch(suffix string) (string, string, bool) {
	token := suffix
	if dot := strings.IndexByte(token, '.'); dot >= 0 {
		token = token[:dot]
	}
	token = strings.TrimPrefix(token, "-")
	if token == "" {
		return "", "", false
	}
	var goos string
	switch {
	case strings.Contains(token, "linux"):
		goos = "linux"
	case strings.Contains(token, "macos"), strings.Contains(token, "darwin"):
		goos = "darwin"
	case strings.Contains(token, "windows"), token == "win64":
		goos = "windows"
	default:
		return "", "", false
	}
	var goarch string
	switch {
	case strings.Contains(token, "arm64"), strings.Contains(token, "aarch64"):
		goarch = "arm64"
	case strings.Contains(token, "x86_64"), strings.Contains(token, "amd64"), strings.Contains(token, "x64"), strings.HasSuffix(token, "64"):
		goarch = "amd64"
	case goos == "darwin" && (token == "macos" || token == "darwin"):
		goarch = "amd64"
	default:
		return "", "", false
	}
	return goos, goarch, true
}

// Portable archives outrank installers. Checksums and other sidecars are rank 0.
func blenderArchiveRank(name string) int {
	switch {
	case strings.HasSuffix(name, ".tar.xz"):
		return 100
	case strings.HasSuffix(name, ".tar.bz2"):
		return 90
	case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"):
		return 80
	case strings.HasSuffix(name, ".zip"):
		return 70
	case strings.HasSuffix(name, ".dmg"):
		return 60
	case strings.HasSuffix(name, ".msi"):
		return 10
	case strings.HasSuffix(name, ".msix"):
		return 5
	case strings.HasSuffix(name, ".exe"):
		return 1
	default:
		return 0
	}
}

func hrefs(body []byte) []string {
	matches := hrefPattern.FindAllSubmatch(body, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, string(match[1]))
	}
	return out
}

func blenderSeriesNames(body []byte) []string {
	var names []string
	seen := map[string]bool{}
	for _, href := range hrefs(body) {
		name := strings.Trim(href, "/")
		if slash := strings.LastIndex(name, "/"); slash >= 0 {
			name = name[slash+1:]
		}
		if !blenderSeriesPattern.MatchString(name) || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

func indexFileNames(body []byte) []string {
	var names []string
	seen := map[string]bool{}
	for _, href := range hrefs(body) {
		if href == "" || strings.HasSuffix(href, "/") {
			continue
		}
		name := path.Base(href)
		if name == "." || name == ".." || name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}
