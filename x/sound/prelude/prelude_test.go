package prelude

import (
	"testing"

	lewfs "github.com/lewtec/lewkit/x/fs"
	gprelude "github.com/lewtec/lewkit/x/generate/prelude"
	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/path/pick"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedMatches(t *testing.T) {
	out := t.TempDir()
	dest := path.New(out).Join("prelude", "prelude.go")
	t.Chdir("..")
	require.NoError(t, gprelude.Run(t.Context(), ".", dest.String()))

	gotFS, err := path.Open(out)
	require.NoError(t, err)
	test.CloseOnCleanup(t, gotFS)
	wantFS, err := path.Open(".")
	require.NoError(t, err)
	test.CloseOnCleanup(t, wantFS)

	for file, err := range lewfs.Walk(t.Context(), gotFS, pick.Glob("**/prelude.go")) {
		require.NoError(t, err)
		if file.Mode.IsDir() {
			continue
		}
		got, err := file.Name.ReadFile(gotFS)
		require.NoError(t, err)
		want, err := file.Name.ReadFile(wantFS)
		require.NoError(t, err)
		assert.Equal(t, string(want), string(got), file.Name.String())
	}
}
