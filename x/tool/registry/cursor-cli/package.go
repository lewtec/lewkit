package pkg

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"

	"strings"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

const (
	cursorInstallScriptURL = "https://cursor.com/install"
	cursorDownloadBaseURL  = "https://downloads.cursor.com/lab"
)

var cursorVersionPattern = regexp.MustCompile(`https://downloads\.cursor\.com/lab/([0-9]{4}\.[0-9]{2}\.[0-9]{2}-[0-9a-fA-F]+)`)

type cursorCLITool struct {
	installScriptURL string
	downloadBaseURL  string
	fetchURL         func(context.Context, string) ([]byte, error)
	goos             string
	goarch           string
}

func newCursorCLI() (tool.Tool, error) {
	return &cursorCLITool{}, nil
}

func (t *cursorCLITool) ListVersions(ctx context.Context) ([]string, error) {
	goos, goarch := install.Host(t.goos, t.goarch)
	if _, _, _, ok := cursorPlatform(goos, goarch); !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrNoPlatformArtifact, goos, goarch)
	}
	body, err := t.fetch(ctx, t.scriptURL())
	if err != nil {
		return nil, err
	}
	return cursorVersions(body)
}

func (t *cursorCLITool) Install(ctx context.Context, version, destDir string) error {
	return install.InstallFirstArtifact(ctx, version, destDir, normalizeCursorVersion, t.ListVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *cursorCLITool) Pin() tool.Pin {
	var pin tool.Pin
	return pin
}

func (t *cursorCLITool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	goos, goarch := install.Host(t.goos, t.goarch)
	osName, archName, archiveName, ok := cursorPlatform(goos, goarch)
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrNoPlatformArtifact, goos, goarch)
	}
	resolved, err := install.ResolveToolVersion(ctx, version, normalizeCursorVersion, t.ListVersions)
	if err != nil {
		return nil, err
	}
	artifactURL := fmt.Sprintf("%s/%s/%s/%s/%s", t.downloadBase(), resolved, osName, archName, archiveName)
	return []tool.Artifact{{
		OS:   goos,
		Arch: goarch,
		URL:  artifactURL,
	}}, nil
}

func (t *cursorCLITool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	if err := install.DefaultInstallArtifact(ctx, artifact, destDir); err != nil {
		return err
	}
	return t.Fix(ctx, destDir)
}

func (t *cursorCLITool) EnsureBinary(ctx context.Context, version, cmdName, destDir string) (string, error) {
	return tool.EnsureBinaryNamed(destDir, cmdName, "cursor-agent", "cursor-cli", func() error {
		return t.Install(ctx, version, destDir)
	}, "cursor-agent", "agent")
}

func (t *cursorCLITool) Fix(ctx context.Context, destDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	root, err := lewpath.Open(destDir)
	if err != nil {
		return err
	}
	defer root.Close()

	unixBinary := lewpath.New("cursor-agent")
	if isFile, err := unixBinary.IsFile(root); err != nil {
		return err
	} else if isFile {
		agentLink := lewpath.New("agent")
		if err := agentLink.Remove(root); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err := root.Symlink(unixBinary.String(), agentLink.String()); err != nil {
			return err
		}
	}

	for _, suffix := range []string{".cmd", ".ps1", ".exe"} {
		source := lewpath.New("cursor-agent" + suffix)
		isFile, err := source.IsFile(root)
		if err != nil {
			return err
		}
		if !isFile {
			continue
		}
		body, err := source.ReadFile(root)
		if err != nil {
			return err
		}
		info, err := source.Stat(root)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o755
		}
		target := lewpath.New("agent" + suffix)
		if err := target.WriteFile(root, body, mode); err != nil {
			return err
		}
	}
	return nil
}

func (t *cursorCLITool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("cursor-agent"), tool.Binary("agent"))
}

func (t *cursorCLITool) scriptURL() string {
	rawURL := strings.TrimSpace(t.installScriptURL)
	if rawURL == "" {
		return cursorInstallScriptURL
	}
	return rawURL
}

func (t *cursorCLITool) downloadBase() string {
	base := strings.TrimSpace(t.downloadBaseURL)
	if base == "" {
		base = cursorDownloadBaseURL
	}
	return strings.TrimRight(base, "/")
}

func (t *cursorCLITool) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	return install.GetBytesOr(ctx, rawURL, t.fetchURL)
}

func normalizeCursorVersion(version string) string {
	return install.NormalizeVPrefixed(version)
}

func cursorVersions(body []byte) ([]string, error) {
	matches := cursorVersionPattern.FindAllSubmatch(body, -1)
	seen := map[string]bool{}
	versions := make([]string, 0, len(matches))
	for _, match := range matches {
		version := string(match[1])
		if version == "" || seen[version] {
			continue
		}
		seen[version] = true
		versions = append(versions, version)
	}
	return install.SortVersionsDesc(versions)
}

func cursorPlatform(goos, goarch string) (osName, archName, archiveName string, ok bool) {
	switch goos {
	case "linux", "android":
		osName = "linux"
		archiveName = "agent-cli-package.tar.gz"
	case "darwin":
		osName = "darwin"
		archiveName = "agent-cli-package.tar.gz"
	case "windows":
		osName = "windows"
		archiveName = "agent-cli-package.zip"
	default:
		return "", "", "", false
	}

	switch goarch {
	case "amd64":
		archName = "x64"
	case "arm64":
		archName = "arm64"
	default:
		return "", "", "", false
	}
	return osName, archName, archiveName, true
}
