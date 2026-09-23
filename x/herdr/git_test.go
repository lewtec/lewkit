package herdr

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitCheckout(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	git("init", "-b", "master")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644))
	git("add", "a.txt")
	git("commit", "-m", "init")

	g := &Git{}
	info, ok := g.Info(t.Context(), dir)
	require.True(t, ok)
	want := dir
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		want = real
	}
	assert.Equal(t, want, info.Toplevel)
	assert.Equal(t, want, info.Root)
	assert.False(t, info.Linked)

	branch, ok := g.Branch(t.Context(), dir)
	require.True(t, ok)
	assert.Equal(t, "master", branch)

	def, ok := g.DefaultBranch(t.Context(), dir)
	require.True(t, ok)
	assert.Equal(t, "master", def)

	require.False(t, g.Dirty(t.Context(), dir))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b\n"), 0o644))
	assert.True(t, g.Dirty(t.Context(), dir))

	rows := g.Worktrees(t.Context(), dir)
	require.NotEmpty(t, rows)
	assert.Equal(t, want, rows[0].Path)
	assert.Equal(t, "master", rows[0].Branch)
}
