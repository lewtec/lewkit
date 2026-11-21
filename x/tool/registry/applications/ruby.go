package applications

import (
	"bytes"
	"context"
	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry"
	"log/slog"
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
	return t.fixRubyShebangs(ctx, destDir)
}

func (t *rubyTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	return ensureToolBinary(ctx, version, cmdName, destDir, "Ruby", t.Install)
}

func (t *rubyTool) Fix(ctx context.Context, destDir string) error {
	return t.fixRubyShebangs(ctx, destDir)
}

// --- helpers (as methods to avoid littering package scope) ---

func (t *rubyTool) normalizeVersion(version string) string {
	return normalizeVersion(version, "ruby-", "Ruby-")
}

func (t *rubyTool) fixRubyShebangs(ctx context.Context, destDir string) error {
	root, err := path.Open(destDir)
	if err != nil {
		return err
	}
	defer root.Close()
	targetRuby := filepath.Join(destDir, "bin", "ruby")
	for name, err := range path.New(".").Walk(ctx, root) {
		if err != nil {
			return err
		}
		isDirectory, err := name.IsDir(root)
		if err != nil || isDirectory {
			continue
		}
		body, err := name.ReadFile(root)
		if err != nil {
			continue
		}
		if !bytes.HasPrefix(body, []byte("#!")) {
			continue
		}
		end := bytes.IndexByte(body, '\n')
		if end == -1 {
			end = len(body)
		}
		shebang := string(body[:end])
		if !strings.Contains(strings.ToLower(shebang), "ruby") {
			continue
		}
		after := strings.TrimPrefix(shebang, "#!")
		cut := len(after)
		for i := 0; i < len(after); i++ {
			if after[i] == ' ' || after[i] == '\t' {
				cut = i
				break
			}
		}
		interpreter := strings.TrimSpace(after[:cut])
		argumentPart := after[cut:]
		if interpreter == targetRuby {
			continue
		}
		base := strings.ToLower(filepath.Base(interpreter))
		if !strings.HasPrefix(base, "ruby") && !strings.Contains(strings.ToLower(interpreter), "ruby") {
			continue
		}
		newShebang := "#!" + targetRuby + argumentPart
		newContent := newShebang + string(body[end:])
		info, err := name.Stat(root)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if mode == 0 {
			mode = 0o755
		}
		if err := name.WriteFile(root, []byte(newContent), mode); err != nil {
			return err
		}
	}
	return nil
}

func (t *rubyTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("ruby"))
}
