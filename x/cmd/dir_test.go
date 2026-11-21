package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/path"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runDirArgCases[T any](t *testing.T, parse func(in string) (T, error), value func(T) string) {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	missing := filepath.Join(dir, "nope")

	cases := []struct {
		name string
		in   string
		want string
		err  error
	}{
		{name: "existing dir", in: dir, want: dir},
		{name: "missing", in: missing, err: os.ErrNotExist},
		{name: "file", in: file, err: path.ErrNotDir},
		{name: "empty", in: "", err: path.ErrEmptyPath},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parse(tc.in)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, value(got))
		})
	}
}
