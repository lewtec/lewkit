package term

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssignAndResolvePath(t *testing.T) {
	path, err := assignPath("", "abcd")
	require.NoError(t, err)
	assert.Equal(t, "abcd", path)

	path, err = assignPath("work/", "abcd")
	require.NoError(t, err)
	assert.Equal(t, "work/abcd", path)

	path, err = assignPath("work/editor", "abcd")
	require.NoError(t, err)
	assert.Equal(t, "work/editor", path)

	_, err = assignPath("../secret", "abcd")
	require.ErrorIs(t, err, ErrPath)

	folder, err := resolvePath("work", "editor")
	require.NoError(t, err)
	assert.Equal(t, "work/editor", folder)

	folder, err = resolvePath("work/editor", "..")
	require.NoError(t, err)
	assert.Equal(t, "work", folder)

	folder, err = resolvePath("work", "/notes")
	require.NoError(t, err)
	assert.Equal(t, "notes", folder)

	_, err = resolvePath("", "..")
	require.ErrorIs(t, err, ErrPath)
}

func TestCatalogFolders(t *testing.T) {
	panes := map[string]*live{
		"a": {info: PaneInfo{ID: "a", Path: "work/editor"}},
		"b": {info: PaneInfo{ID: "b", Path: "work/docs"}},
		"c": {info: PaneInfo{ID: "c", Path: "scratch"}},
	}
	entries := catalog([]string{"a", "b", "c"}, panes)
	var dirs []string
	var files []string
	for _, entry := range entries {
		if entry.Dir {
			dirs = append(dirs, entry.Path)
			continue
		}
		files = append(files, entry.Path)
	}
	assert.Equal(t, []string{"work"}, dirs)
	assert.Equal(t, []string{"scratch", "work/docs", "work/editor"}, files)

	children := directChildren(entries, "work")
	require.Len(t, children, 2)
	assert.Equal(t, "work/docs", children[0].Path)
	assert.False(t, children[0].Dir)

	assert.True(t, pathTaken(map[string]string{"work/editor": "a"}, "work"))
	assert.False(t, pathTaken(map[string]string{"work/editor": "a"}, "notes"))
	assert.True(t, pathUnder("work/editor", "work"))
	assert.False(t, pathUnder("workshop", "work"))
}
