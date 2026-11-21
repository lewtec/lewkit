package applications

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/httpclient"
	"github.com/lewtec/lewkit/x/tool"
	"github.com/lewtec/lewkit/x/tool/github"
	"github.com/lewtec/lewkit/x/tool/registry"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

func init() {
	registry.RegisterTool("llvm", newLLVM)
}

type llvmTool struct {
	inner tool.Tool
}

func newLLVM() (tool.Tool, error) {
	inner, err := github.NewTool("llvm/llvm-project")
	if err != nil {
		return nil, err
	}
	return &llvmTool{inner: inner}, nil
}

func (t *llvmTool) ListVersions(ctx context.Context) ([]string, error) {
	// Use a direct fetch with larger page size so older releases (e.g. 18.x)
	// remain visible in the registry. The generic github backend only gets the
	// default first page (~30).
	u := "https://api.github.com/repos/llvm/llvm-project/releases?per_page=100"
	req, err := github.NewAPIRequest(ctx, http.MethodGet, u)
	if err != nil {
		return nil, err
	}

	httpDriver, err := driver.Get[httpclient.Driver](ctx)
	if err != nil {
		return nil, err
	}
	resp, err := httpDriver.Client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("github releases for llvm: %s (reading body: %v)", resp.Status, err)
		}
		msg := strings.TrimSpace(string(body))
		if resp.StatusCode == http.StatusForbidden && strings.Contains(msg, "rate limit") {
			return nil, fmt.Errorf("github api rate limit exceeded for llvm releases (consider setting GITHUB_TOKEN or 'gh auth login')")
		}
		if msg != "" {
			return nil, fmt.Errorf("github releases for llvm: %s: %s", resp.Status, msg)
		}
		return nil, fmt.Errorf("github releases for llvm: %s", resp.Status)
	}

	var rels []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rels); err != nil {
		return nil, err
	}

	out := make([]string, 0, len(rels))
	for _, r := range rels {
		v := strings.TrimSpace(r.TagName)
		if !strings.HasPrefix(v, "llvmorg-") {
			continue
		}
		ver := strings.TrimPrefix(v, "llvmorg-")
		if ver == "" || strings.Contains(ver, "-") {
			continue
		}
		out = append(out, ver)
	}
	if len(out) == 0 {
		return nil, ErrNoVersions
	}
	return out, nil
}

func (t *llvmTool) Install(ctx context.Context, version string, destDir string) error {
	slog.WarnContext(ctx, "llvm (registry backend) is experimental; backed by official clang+llvm prebuilts from llvm/llvm-project")

	v := strings.TrimSpace(version)
	if v == "" || v == "latest" {
		return t.inner.Install(ctx, v, destDir)
	}
	tag := "llvmorg-" + normalizeLLVMVersion(v)
	return t.inner.Install(ctx, tag, destDir)
}

func (t *llvmTool) Pin() tool.Pin {
	var pin tool.Pin
	// LLVM prebuilts come from llvm/llvm-project GitHub releases under llvmorg-* tags.

	return pin
}

func (t *llvmTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	tag := toLLVMSpecTag(version)
	if at, ok := t.inner.(interface {
		ListArtifacts(context.Context, string) ([]tool.Artifact, error)
	}); ok {
		return at.ListArtifacts(ctx, tag)
	}
	return nil, ErrNoPlatformArtifact
}

func (t *llvmTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destDir string) error {
	slog.WarnContext(ctx, "llvm (registry backend) is experimental; backed by official clang+llvm prebuilts from llvm/llvm-project")

	// Delegate to inner when possible so GitHub auth/asset download logic is used.
	if at, ok := t.inner.(interface {
		InstallArtifact(context.Context, tool.Artifact, string) error
	}); ok {
		return at.InstallArtifact(ctx, artifact, destDir)
	}
	return defaultInstallArtifact(ctx, artifact, destDir)
}

func (t *llvmTool) EnsureBinary(ctx context.Context, version string, cmdName string, destDir string) (string, error) {
	return ensureToolBinary(ctx, version, cmdName, destDir, "LLVM", t.Install)
}

// --- helpers ---

func normalizeLLVMVersion(version string) string {
	return normalizeVersion(version, "llvmorg-", "v", "V")
}

func toLLVMSpecTag(version string) string {
	v := strings.TrimSpace(version)
	if v == "" || v == "latest" {
		return v
	}
	n := normalizeLLVMVersion(v)
	if n == "" {
		return v
	}
	return "llvmorg-" + n
}

func (t *llvmTool) InstallChecks() []tool.Check {
	return tool.Checks(tool.Binary("clang"))
}
