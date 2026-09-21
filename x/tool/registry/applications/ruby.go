package applications

import (
	"bytes"
	"context"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	registry.RegisterTool("ruby", newRuby)
}

type rubyTool struct {
	inner tool.Tool
}

func newRuby() (tool.Tool, error) {
	inner, err := github.NewTool("ruby/ruby-builder")
	if err != nil {
		return nil, err
	}
	return &rubyTool{inner: inner}, nil
}

func (t *rubyTool) ListVersions(ctx context.Context) ([]string, error) {
	vers, err := t.inner.ListVersions(ctx)
	if err != nil {
		return nil, err
	}

	out := []string{}
	seen := map[string]bool{}
	for _, v := range vers {
		if !strings.HasPrefix(v, "ruby-") {
			continue
		}
		ver := strings.TrimPrefix(v, "ruby-")
		if ver == "" || strings.Contains(ver, "-") || seen[ver] {
			continue
		}
		seen[ver] = true
		out = append(out, ver)
	}
	return sortVersionsDesc(out)
}

func (t *rubyTool) Install(ctx context.Context, version string, destDir string) error {
	return installSelectedArtifact(ctx, version, destDir, "ruby", "registry:ruby", t.normalizeVersion, t.ListVersions, t.ListArtifacts, t.InstallArtifact)
}

func (t *rubyTool) Pin() tool.Pin {
	var pin tool.Pin
	pin.Versioning = "semver"

	return pin
}

func (t *rubyTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	v, err := resolveToolVersion(ctx, version, t.normalizeVersion, t.ListVersions)
	if err != nil {
		return nil, err
	}

	tag := "ruby-" + v
	at, ok := t.inner.(tool.ArtifactTool)
	if !ok {
		return nil, registry.ErrNoArtifactTool
	}
	return at.ListArtifacts(ctx, tag)
}

func (t *rubyTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	slog.WarnContext(ctx, "ruby (registry backend) is experimental; backed by ruby/ruby-builder prebuilts")

	if err := tool.InstallArtifact(ctx, artifact, destDir, tool.DownloadOptions{}); err != nil {
		return err
	}
	return t.fixRubyShebangs(destDir)
}

func (t *rubyTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	return ensureToolBinary(ctx, version, cmdName, destDir, "Ruby", t.Install)
}

func (t *rubyTool) Fix(_ context.Context, destDir string) error {
	return t.fixRubyShebangs(destDir)
}

// --- helpers (as methods to avoid littering package scope) ---

func (t *rubyTool) normalizeVersion(version string) string {
	return normalizeVersion(version, "ruby-", "Ruby-")
}

func (t *rubyTool) fixRubyShebangs(destDir string) error {
	targetRuby := filepath.Join(destDir, "bin", "ruby")
	return filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil // skip unreadable
		}
		if !bytes.HasPrefix(b, []byte("#!")) {
			return nil
		}
		// locate end of shebang line
		end := bytes.IndexByte(b, '\n')
		if end == -1 {
			end = len(b)
		}
		shebang := string(b[:end])
		if !strings.Contains(strings.ToLower(shebang), "ruby") {
			return nil
		}
		after := strings.TrimPrefix(shebang, "#!")
		// split at first whitespace to separate interpreter from args
		cut := len(after)
		for i := 0; i < len(after); i++ {
			if after[i] == ' ' || after[i] == '\t' {
				cut = i
				break
			}
		}
		interp := strings.TrimSpace(after[:cut])
		argPart := after[cut:]
		if interp == targetRuby {
			return nil
		}
		// only rewrite if it refers to a ruby interpreter
		base := strings.ToLower(filepath.Base(interp))
		if !strings.HasPrefix(base, "ruby") && !strings.Contains(strings.ToLower(interp), "ruby") {
			return nil
		}
		newShebang := "#!" + targetRuby + argPart
		newContent := newShebang + string(b[end:])
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o755
		}
		return os.WriteFile(path, []byte(newContent), mode)
	})
}

func (t *rubyTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("ruby"))
}
