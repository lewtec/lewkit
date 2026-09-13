package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataDirArg(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	missing := filepath.Join(dir, "nope")

	type args struct {
		dir DataDirArg `long:"dir"`
	}
	cases := []struct {
		name string
		in   string
		want string
		err  error
	}{
		{name: "existing dir", in: dir, want: dir},
		{name: "missing", in: missing, err: os.ErrNotExist},
		{name: "file", in: file, err: ErrNotDir},
		{name: "empty", in: "", err: ErrEmptyPath},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse[args]("--dir", tc.in)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.dir.Value())
		})
	}
}

func TestDataDirDefault(t *testing.T) {
	type args struct {
		dir DataDirArg `long:"dir" default:"."`
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	assert.Equal(t, ".", got.dir.Value())
}
