package term

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitSummaryReadsPorcelain(t *testing.T) {
	assert.Equal(t, "no history here", gitSummary(""))
	assert.Equal(t, "no history here", gitSummary("not a repo\n"))
	assert.Equal(t, "main · tidy", gitSummary("## main\n"))
	assert.Equal(t, "main · tidy", gitSummary("## main...origin/main\n"))
	assert.Equal(t, "main · tidy", gitSummary("## No commits yet on main\n"))
	assert.Equal(t, "main · 1 changed", gitSummary("## main\n M file\n"))
	assert.Equal(t, "main · 2 changed", gitSummary("## main\n M a\n?? b\n"))
	assert.Equal(t, "detached · tidy", gitSummary("## HEAD (no branch)\n"))
	assert.Equal(t, "detached · tidy", gitSummary("## HEAD\n"))
}

func TestRoomLabelUsesHome(t *testing.T) {
	assert.Equal(t, "nowhere yet", roomLabel(""))
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	require.NotEmpty(t, home)
	assert.Equal(t, "~", roomLabel(home))
	assert.Equal(t, "~/drawers", roomLabel(filepath.Join(home, "drawers")))
	outside := "/lewkit-term-outside"
	if outside == home || strings.HasPrefix(outside, home+"/") {
		outside = "/var/lewkit-term-outside"
	}
	assert.Equal(t, outside, roomLabel(outside))
}

func TestReadGitUsesTheDirectory(t *testing.T) {
	test.Need(t, "git")
	dir := t.TempDir()
	assert.Equal(t, "no history here", readGit(t.Context(), dir))
	assert.Equal(t, "no history here", readGit(t.Context(), ""))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.Equal(t, "no history here", readGit(ctx, dir))

	require.NoError(t, exec.Command("git", "init", "-b", "main", dir).Run())
	assert.Equal(t, "main · tidy", readGit(t.Context(), dir))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "note"), []byte("x"), 0o644))
	assert.Equal(t, "main · 1 changed", readGit(t.Context(), dir))
}
