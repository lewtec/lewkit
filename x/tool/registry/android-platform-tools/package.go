package androidplatformtools

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

// androidPlatformToolsIndexURL is the SDK repository that lists the current
// platform-tools revision, archive names, and checksums.
const androidPlatformToolsIndexURL = "https://dl.google.com/android/repository/repository2-1.xml"

// androidPlatformToolsBaseURL resolves relative archive names from that index.
const androidPlatformToolsBaseURL = "https://dl.google.com/android/repository/"

type androidPlatformToolsTool struct {
	indexURL string
	fetchURL func(context.Context, string) ([]byte, error)
	goos     string
	goarch   string
}

func newAndroidPlatformTools() (tool.Tool, error) {
	return &androidPlatformToolsTool{indexURL: androidPlatformToolsIndexURL}, nil
}

func (t *androidPlatformToolsTool) host() (string, string) {
	goos := t.goos
	goarch := t.goarch
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	return goos, goarch
}

func (t *androidPlatformToolsTool) ListVersions(ctx context.Context) ([]string, error) {
	goos, goarch := t.host()
	builds, err := t.builds(ctx, goos, goarch)
	if err != nil {
		return nil, err
	}
	return platformToolsVersions(builds)
}

func (t *androidPlatformToolsTool) Install(ctx context.Context, version, destDir string) error {
	artifacts, err := t.ListArtifacts(ctx, version)
	if err != nil {
		return err
	}
	return t.InstallArtifact(ctx, artifacts[0], destDir)
}

func (t *androidPlatformToolsTool) Pin() tool.Pin {
	var pin tool.Pin
	pin.Versioning = "semver"
	// repository2-1.xml is not a Renovate datasource. The checksum travels
	// with the archive entry and is checked at download time.

	return pin
}

func (t *androidPlatformToolsTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	goos, goarch := t.host()
	builds, err := t.builds(ctx, goos, goarch)
	if err != nil {
		return nil, err
	}
	if len(builds) == 0 {
		return nil, install.ErrNoPlatformArtifact
	}
	want := normalizePlatformToolsVersion(version)
	if want == "" || want == "latest" {
		versions, err := platformToolsVersions(builds)
		if err != nil {
			return nil, err
		}
		want = versions[0]
	}
	for _, build := range builds {
		if build.Version != want {
			continue
		}
		return []tool.Artifact{{
			OS:   goos,
			Arch: goarch,
			URL:  build.URL,
			Hash: build.Hash,
			Size: build.Size,
		}}, nil
	}
	return nil, install.ErrNoPlatformArtifact
}

func (t *androidPlatformToolsTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	return install.DefaultInstallArtifact(ctx, artifact, destDir)
}

func (t *androidPlatformToolsTool) EnsureBinary(ctx context.Context, version, cmdName, destDir string) (string, error) {
	return install.EnsureToolBinary(ctx, version, cmdName, destDir, "Android platform-tools", t.Install)
}

func (t *androidPlatformToolsTool) InstallChecks() []tool.Check {
	names := []string{
		"adb",
		"fastboot",
		"etc1tool",
		"hprof-conv",
		"make_f2fs",
		"make_f2fs_casefold",
		"mke2fs",
		"sqlite3",
	}
	checks := make([]tool.Check, len(names))
	for i, name := range names {
		checks[i] = tool.Binary(name)
	}
	return checks
}

func (t *androidPlatformToolsTool) builds(ctx context.Context, goos, goarch string) ([]platformToolsBuild, error) {
	index := t.indexURL
	if index == "" {
		index = androidPlatformToolsIndexURL
	}
	var body []byte
	var err error
	if t.fetchURL != nil {
		body, err = t.fetchURL(ctx, index)
	} else {
		body, err = install.GetBytes(ctx, index)
	}
	if err != nil {
		return nil, err
	}
	return platformToolsBuilds(body, goos, goarch)
}

func platformToolsVersions(builds []platformToolsBuild) ([]string, error) {
	seen := map[string]bool{}
	versions := make([]string, 0, len(builds))
	for _, build := range builds {
		if build.Version == "" || seen[build.Version] {
			continue
		}
		seen[build.Version] = true
		versions = append(versions, build.Version)
	}
	return install.SortVersionsDesc(versions)
}

type platformToolsBuild struct {
	Version string
	Arch    string
	URL     string
	Hash    string
	Size    int64
}

type platformToolsRepository struct {
	Packages []platformToolsPackage `xml:"remotePackage"`
}

type platformToolsPackage struct {
	Path     string                  `xml:"path,attr"`
	Revision platformToolsRevision   `xml:"revision"`
	Archives platformToolsArchiveSet `xml:"archives"`
}

type platformToolsRevision struct {
	Major int `xml:"major"`
	Minor int `xml:"minor"`
	Micro int `xml:"micro"`
}

func (rev platformToolsRevision) String() string {
	return strconv.Itoa(rev.Major) + "." + strconv.Itoa(rev.Minor) + "." + strconv.Itoa(rev.Micro)
}

type platformToolsArchiveSet struct {
	Archive []platformToolsArchive `xml:"archive"`
}

type platformToolsArchive struct {
	Complete platformToolsComplete `xml:"complete"`
	HostOS   string                `xml:"host-os"`
	HostArch string                `xml:"host-arch"`
	HostBits string                `xml:"host-bits"`
}

type platformToolsComplete struct {
	Size     int64                 `xml:"size"`
	Checksum platformToolsChecksum `xml:"checksum"`
	URL      string                `xml:"url"`
}

type platformToolsChecksum struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

func platformToolsBuilds(body []byte, goos, goarch string) ([]platformToolsBuild, error) {
	var doc platformToolsRepository
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("android platform-tools index: %w", err)
	}
	wantOS := platformToolsHostOS(goos)
	chosen := map[string]platformToolsBuild{}
	for _, pkg := range doc.Packages {
		if pkg.Path != "platform-tools" || pkg.Revision.Major == 0 {
			continue
		}
		version := pkg.Revision.String()
		for _, archive := range pkg.Archives.Archive {
			if !platformToolsHostMatch(archive.HostOS, wantOS) {
				continue
			}
			downloadURL, err := platformToolsDownloadURL(archive.Complete.URL)
			if err != nil {
				return nil, err
			}
			if downloadURL == "" {
				continue
			}
			hash := formatPlatformToolsChecksum(archive.Complete.Checksum)
			if hash == "" {
				continue
			}
			build := platformToolsBuild{
				Version: version,
				URL:     downloadURL,
				Hash:    hash,
				Size:    archive.Complete.Size,
			}
			for _, arch := range platformToolsArchs(archive, downloadURL) {
				if arch != goarch {
					continue
				}
				build.Arch = arch
				if prev, ok := chosen[version]; ok {
					chosen[version] = preferPlatformToolsBuild(prev, build)
					continue
				}
				chosen[version] = build
			}
		}
	}
	if len(chosen) == 0 {
		return nil, nil
	}
	out := make([]platformToolsBuild, 0, len(chosen))
	for _, build := range chosen {
		out = append(out, build)
	}
	return out, nil
}

func platformToolsHostOS(goos string) string {
	if goos == "darwin" {
		return "macosx"
	}
	return goos
}

func platformToolsHostMatch(archiveOS, wantOS string) bool {
	got := strings.ToLower(strings.TrimSpace(archiveOS))
	if got == "" {
		return true
	}
	return got == wantOS
}

// platformToolsArchs maps one repository archive onto Go arch names.
// The index omits host-arch. The darwin zip is a universal binary. The linux
// and windows zips are x86_64 unless the file name or host-arch says otherwise.
func platformToolsArchs(archive platformToolsArchive, downloadURL string) []string {
	if arch := normalizePlatformToolsArch(archive.HostArch); arch != "" {
		return filterPlatformToolsBits([]string{arch}, archive.HostBits)
	}
	if named := platformToolsFilenameArchs(downloadURL); len(named) > 0 {
		return filterPlatformToolsBits(named, archive.HostBits)
	}
	return filterPlatformToolsBits(defaultPlatformToolsArchs(archive.HostOS, archive.HostBits), archive.HostBits)
}

func defaultPlatformToolsArchs(hostOS, bits string) []string {
	switch strings.ToLower(strings.TrimSpace(hostOS)) {
	case "macosx", "darwin":
		return []string{"amd64", "arm64"}
	default:
		if strings.TrimSpace(bits) == "32" {
			return []string{"386"}
		}
		return []string{"amd64"}
	}
}

func filterPlatformToolsBits(archs []string, bits string) []string {
	switch strings.TrimSpace(bits) {
	case "32":
		var out []string
		for _, arch := range archs {
			if arch == "386" {
				out = append(out, arch)
			}
		}
		return out
	case "64":
		var out []string
		for _, arch := range archs {
			if arch != "386" {
				out = append(out, arch)
			}
		}
		return out
	default:
		return archs
	}
}

func normalizePlatformToolsArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "x64", "x86_64", "amd64":
		return "amd64"
	case "arm64", "aarch64":
		return "arm64"
	case "x86", "i386", "ia32":
		return "386"
	default:
		return ""
	}
}

func platformToolsFilenameArchs(downloadURL string) []string {
	name := strings.ToLower(path.Base(downloadURL))
	var archs []string
	if strings.Contains(name, "arm64") || strings.Contains(name, "aarch64") {
		archs = append(archs, "arm64")
	}
	if strings.Contains(name, "x86_64") || strings.Contains(name, "amd64") || strings.Contains(name, "-x64") || strings.Contains(name, "_x64") {
		archs = append(archs, "amd64")
	}
	if !strings.Contains(name, "x86_64") && (strings.Contains(name, "i386") || strings.Contains(name, "-x86") || strings.Contains(name, "_x86")) {
		archs = append(archs, "386")
	}
	return archs
}

func preferPlatformToolsBuild(current, next platformToolsBuild) platformToolsBuild {
	if slices.Contains(platformToolsFilenameArchs(next.URL), next.Arch) && !slices.Contains(platformToolsFilenameArchs(current.URL), current.Arch) {
		return next
	}
	return current
}

func platformToolsDownloadURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("platform-tools archive url: %w", err)
	}
	if parsed.IsAbs() {
		return parsed.String(), nil
	}
	base, err := url.Parse(androidPlatformToolsBaseURL)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(parsed).String(), nil
}

func formatPlatformToolsChecksum(sum platformToolsChecksum) string {
	value := strings.ToLower(strings.TrimSpace(sum.Value))
	algo := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(sum.Type)), "-", "")
	if algo == "" {
		switch len(value) {
		case 40:
			algo = "sha1"
		case 64:
			algo = "sha256"
		default:
			return ""
		}
	}
	switch algo {
	case "sha1":
		if len(value) != 40 {
			return ""
		}
	case "sha256":
		if len(value) != 64 {
			return ""
		}
	default:
		return ""
	}
	return algo + ":" + value
}

func normalizePlatformToolsVersion(version string) string {
	return install.NormalizeVersion(version, "r", "R", "v", "V")
}
