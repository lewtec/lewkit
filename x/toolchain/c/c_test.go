package c

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoHeader(t *testing.T) {
	_, _, err := Go(nil, "ping.h")
	require.ErrorIs(t, err, ErrNilContext)
	dir := t.TempDir()
	file := filepath.Join(dir, "ping.h")
	body := "#define ANSWER 7\nint ping(int x);\n"
	require.NoError(t, os.WriteFile(file, []byte(body), 0o644))
	path, src, err := Go(t.Context(), file)
	require.NoError(t, err)
	if path != "ping" {
		require.FailNowf(t, "", "import path %s", path)
	}
	if !strings.Contains(src, "package ping") || !strings.Contains(src, "ANSWER") || !strings.Contains(src, "ping") {
		require.FailNowf(t, "", "source:\n%s", src)
	}
}

func TestGoRejectsPackageName(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "foo-bar.h")
	require.NoError(t, os.WriteFile(file, []byte("int x;\n"), 0o644))
	_, _, err := Go(t.Context(), file)
	require.ErrorIs(t, err, ErrName)
}
