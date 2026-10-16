package ignore_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/fs/ignore"
	lewpath "github.com/lewtec/lewkit/x/path"
)

func TestCollectNilAndNotDirectory(t *testing.T) {
	_, err := ignore.Collect(t.Context(), nil, ignore.Config{})
	require.ErrorIs(t, err, errs.ErrMissing)

	_, err = ignore.Collect(t.Context(), fileFS{}, ignore.Config{})
	require.ErrorIs(t, err, lewpath.ErrNotDir)
}

func TestCollectCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := ignore.Collect(ctx, fstest.MapFS{}, ignore.Config{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestGitignoreNestedNegation(t *testing.T) {
	tree := fstest.MapFS{
		".gitignore":     {Data: []byte("*.tmp\n")},
		"pkg/.gitignore": {Data: []byte("!keep.tmp\n")},
		"a.tmp":          {Data: []byte("x")},
		"pkg/keep.tmp":   {Data: []byte("kept")},
		"pkg/drop.tmp":   {Data: []byte("x")},
	}
	fsys := collect(t, tree)

	require.True(t, fsys.Ignored("a.tmp"))
	require.False(t, fsys.Ignored("pkg/keep.tmp"))
	require.True(t, fsys.Ignored("pkg/drop.tmp"))
	require.False(t, fsys.SkipDir("pkg"))

	body, err := fs.ReadFile(fsys, "pkg/keep.tmp")
	require.NoError(t, err)
	require.Equal(t, "kept", string(body))
	_, err = fs.ReadFile(fsys, "a.tmp")
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.ElementsMatch(t, []string{"keep.tmp", ".gitignore"}, dirNames(t, fsys, "pkg"))

	sub, err := fs.Sub(fsys, "pkg")
	require.NoError(t, err)
	body, err = fs.ReadFile(sub, "keep.tmp")
	require.NoError(t, err)
	require.Equal(t, "kept", string(body))
	_, err = fs.ReadFile(sub, "drop.tmp")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestParentDirectoryRuleIsNotUndone(t *testing.T) {
	tree := fstest.MapFS{
		".gitignore":    {Data: []byte("build/\n!build/keep.go\n")},
		"build/a.go":    {Data: []byte("package b\n")},
		"build/keep.go": {Data: []byte("package b\n")},
		"main.go":       {Data: []byte("package main\n")},
	}
	fsys := collect(t, tree)

	require.True(t, fsys.Ignored("build"))
	require.True(t, fsys.Ignored("build/a.go"))
	require.True(t, fsys.Ignored("build/keep.go"))
	require.True(t, fsys.SkipDir("build"))
	require.NotContains(t, walk(t, fsys), "build")
	require.NotContains(t, walk(t, fsys), "build/keep.go")
	require.Contains(t, walk(t, fsys), "main.go")
	_, err := fs.Stat(fsys, "build")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestContentsNegationStillApplies(t *testing.T) {
	tree := fstest.MapFS{
		".gitignore":    {Data: []byte("build/*\n!build/keep.go\n")},
		"build/a.go":    {Data: []byte("package b\n")},
		"build/keep.go": {Data: []byte("package keep\n")},
	}
	fsys := collect(t, tree)

	require.False(t, fsys.Ignored("build"))
	require.True(t, fsys.Ignored("build/a.go"))
	require.False(t, fsys.Ignored("build/keep.go"))
	require.False(t, fsys.SkipDir("build"))

	names := walk(t, fsys)
	require.Contains(t, names, "build")
	require.Contains(t, names, "build/keep.go")
	require.NotContains(t, names, "build/a.go")
	body, err := fs.ReadFile(fsys, "build/keep.go")
	require.NoError(t, err)
	require.Equal(t, "package keep\n", string(body))
}

func TestStarStarAllowsNegationInside(t *testing.T) {
	tree := fstest.MapFS{
		".gitignore":  {Data: []byte("gen/**\n!gen/keep.go\n")},
		"gen/a.go":    {Data: []byte("package g\n")},
		"gen/keep.go": {Data: []byte("package g\n")},
	}
	fsys := collect(t, tree)
	require.True(t, fsys.Ignored("gen/a.go"))
	require.False(t, fsys.Ignored("gen/keep.go"))
	require.False(t, fsys.SkipDir("gen"))
	require.Contains(t, walk(t, fsys), "gen/keep.go")
	require.NotContains(t, walk(t, fsys), "gen/a.go")

	closed := collect(t, fstest.MapFS{
		".gitignore": {Data: []byte("gen/**\n")},
		"gen/a.go":   {Data: []byte("package g\n")},
	})
	require.True(t, closed.SkipDir("gen"))
	require.NotContains(t, walk(t, closed), "gen")
}

func TestExcludedDirectoryBlocksChildNegation(t *testing.T) {
	tree := fstest.MapFS{
		".gitignore":   {Data: []byte("*\n!keep.go\n")},
		"keep.go":      {Data: []byte("package k\n")},
		"sub/keep.go":  {Data: []byte("package k\n")},
		"sub/other.go": {Data: []byte("package k\n")},
	}
	fsys := collect(t, tree)
	require.False(t, fsys.Ignored("keep.go"))
	require.True(t, fsys.Ignored("sub/keep.go"))
	require.True(t, fsys.Ignored("sub/other.go"))
	require.True(t, fsys.SkipDir("sub"))
	require.Equal(t, []string{".", "keep.go"}, walk(t, fsys))
}

func TestReincludeDirectoryThenFile(t *testing.T) {
	tree := fstest.MapFS{
		".gitignore":  {Data: []byte("/*\n!/keep/\n/keep/*\n!/keep/yes.go\n")},
		"keep/yes.go": {Data: []byte("package k\n")},
		"keep/no.go":  {Data: []byte("package k\n")},
	}
	fsys := collect(t, tree)
	require.False(t, fsys.Ignored("keep"))
	require.False(t, fsys.Ignored("keep/yes.go"))
	require.True(t, fsys.Ignored("keep/no.go"))
	require.False(t, fsys.SkipDir("keep"))
	require.Contains(t, walk(t, fsys), "keep/yes.go")
	require.NotContains(t, walk(t, fsys), "keep/no.go")
}

func TestDeeperGitignoreWins(t *testing.T) {
	tree := fstest.MapFS{
		".gitignore":     {Data: []byte("*.tmp\n")},
		"a/.gitignore":   {Data: []byte("!*.tmp\n")},
		"a/b/.gitignore": {Data: []byte("*.tmp\n")},
		"a/y.tmp":        {Data: []byte("x")},
		"a/b/x.tmp":      {Data: []byte("x")},
	}
	fsys := collect(t, tree)
	require.False(t, fsys.Ignored("a/y.tmp"))
	require.True(t, fsys.Ignored("a/b/x.tmp"))
	require.Contains(t, walk(t, fsys), "a/y.tmp")
	require.NotContains(t, walk(t, fsys), "a/b/x.tmp")
}

func TestExcludeThenGitignore(t *testing.T) {
	tree := fstest.MapFS{
		".git/info/exclude": {Data: []byte("secret.txt\n*.log\n")},
		".gitignore":        {Data: []byte("!secret.txt\n")},
		"secret.txt":        {Data: []byte("x")},
		"debug.log":         {Data: []byte("x")},
	}
	fsys := collect(t, tree)
	require.False(t, fsys.Ignored("secret.txt"))
	require.True(t, fsys.Ignored("debug.log"))
	require.Contains(t, walk(t, fsys), "secret.txt")
	require.NotContains(t, walk(t, fsys), "debug.log")
}

func TestParentThenNestedGitignore(t *testing.T) {
	tree := fstest.MapFS{
		".git/info/exclude": {Data: []byte("hidden.txt\n")},
		".gitignore":        {Data: []byte("*.log\n")},
		"svc/.gitignore":    {Data: []byte("!keep.log\n")},
		"svc/keep.log":      {Data: []byte("x")},
		"svc/drop.log":      {Data: []byte("x")},
		"svc/hidden.txt":    {Data: []byte("x")},
		"outside.log":       {Data: []byte("x")},
	}
	fsys := collect(t, tree)
	require.False(t, fsys.Ignored("svc/keep.log"))
	require.True(t, fsys.Ignored("svc/drop.log"))
	require.True(t, fsys.Ignored("svc/hidden.txt"))
	require.True(t, fsys.Ignored("outside.log"))
}

func TestGitdirInsideFS(t *testing.T) {
	tree := fstest.MapFS{
		".git":                {Data: []byte("gitdir: wt\n")},
		"wt/commondir":        {Data: []byte("../common\n")},
		"common/info/exclude": {Data: []byte("from-common.txt\n")},
		"from-common.txt":     {Data: []byte("x")},
		"keep.txt":            {Data: []byte("x")},
	}
	fsys := collect(t, tree)
	require.True(t, fsys.Ignored("from-common.txt"))
	require.False(t, fsys.Ignored("keep.txt"))

	escaped := collect(t, fstest.MapFS{
		".git":     {Data: []byte("gitdir: /outside/repo.git\n")},
		"note.txt": {Data: []byte("x")},
	})
	require.False(t, escaped.Ignored("note.txt"))
}

func TestSkipDirsAreOptional(t *testing.T) {
	tree := fstest.MapFS{
		"node_modules/x/y.go": {Data: []byte("package x\n")},
		"pkg/y.go":            {Data: []byte("package p\n")},
	}
	plain := collect(t, tree)
	require.False(t, plain.Ignored("node_modules/x/y.go"))
	require.False(t, plain.SkipDir("node_modules"))
	require.Contains(t, walk(t, plain), "node_modules/x/y.go")

	fsys, err := ignore.Collect(t.Context(), tree, ignore.Config{SkipDirs: []string{"node_modules", "node_modules", ""}})
	require.NoError(t, err)
	require.True(t, fsys.Ignored("node_modules/x/y.go"))
	require.True(t, fsys.SkipDir("node_modules"))
	require.False(t, fsys.Ignored("pkg/y.go"))
	require.NotContains(t, walk(t, fsys), "node_modules")
	require.Contains(t, walk(t, fsys), "pkg/y.go")
}

func TestAttributesAreOptIn(t *testing.T) {
	tree := fstest.MapFS{
		".gitattributes": {Data: []byte("" +
			"*.pb.go linguist-generated=true\n" +
			"gen/** linguist-generated\n" +
			"gen/keep.go -linguist-generated\n" +
			"*.snap refactree-ignored\n" +
			"hand.go linguist-generated=false\n" +
			"*.go linguist-language=Go\n")},
		"api/foo.pb.go": {Data: []byte("package api\n")},
		"api/hand.go":   {Data: []byte("package api\n")},
		"gen/a.go":      {Data: []byte("package gen\n")},
		"gen/keep.go":   {Data: []byte("package gen\n")},
		"x.snap":        {Data: []byte("snap")},
		"hand.go":       {Data: []byte("package main\n")},
	}
	plain := collect(t, tree)
	require.False(t, plain.Ignored("api/foo.pb.go"))
	require.False(t, plain.Ignored("x.snap"))

	fsys, err := ignore.Collect(t.Context(), tree, ignore.Config{Attributes: true})
	require.NoError(t, err)
	require.True(t, fsys.Ignored("api/foo.pb.go"))
	require.False(t, fsys.Ignored("api/hand.go"))
	require.True(t, fsys.Ignored("gen/a.go"))
	require.False(t, fsys.Ignored("gen/keep.go"))
	require.False(t, fsys.SkipDir("gen"))
	require.True(t, fsys.Ignored("x.snap"))
	require.False(t, fsys.Ignored("hand.go"))
	require.Contains(t, walk(t, fsys), "gen/keep.go")
	require.NotContains(t, walk(t, fsys), "api/foo.pb.go")
	require.NotContains(t, walk(t, fsys), "x.snap")
}

func TestAttributeDirectoryRuleIsNotUndone(t *testing.T) {
	tree := fstest.MapFS{
		".gitattributes": {Data: []byte("dist/ linguist-generated\ndist/keep.go -linguist-generated\n")},
		"dist/a.go":      {Data: []byte("package d\n")},
		"dist/keep.go":   {Data: []byte("package d\n")},
	}
	fsys, err := ignore.Collect(t.Context(), tree, ignore.Config{Attributes: true})
	require.NoError(t, err)
	require.True(t, fsys.Ignored("dist/a.go"))
	require.True(t, fsys.Ignored("dist/keep.go"))
	require.True(t, fsys.SkipDir("dist"))
	require.NotContains(t, walk(t, fsys), "dist")
}

func TestNestedAttributes(t *testing.T) {
	tree := fstest.MapFS{
		"pkg/.gitattributes": {Data: []byte("*.gen.go linguist-generated\n")},
		"pkg/a.gen.go":       {Data: []byte("package p\n")},
		"pkg/a.go":           {Data: []byte("package p\n")},
	}
	fsys, err := ignore.Collect(t.Context(), tree, ignore.Config{Attributes: true})
	require.NoError(t, err)
	require.True(t, fsys.Ignored("pkg/a.gen.go"))
	require.False(t, fsys.Ignored("pkg/a.go"))
}

func TestAnchoredDirectoryAndComments(t *testing.T) {
	tree := fstest.MapFS{
		".gitignore":           {Data: []byte("# *.go\n\n/public/\n")},
		"public/index.html":    {Data: []byte("<html>")},
		"public/en/index.html": {Data: []byte("<html>")},
		"src/page.go":          {Data: []byte("package src\n")},
	}
	fsys := collect(t, tree)
	require.True(t, fsys.Ignored("public/index.html"))
	require.True(t, fsys.Ignored("public/en/index.html"))
	require.True(t, fsys.SkipDir("public"))
	require.False(t, fsys.Ignored("src/page.go"))
	require.False(t, fsys.Ignored("missing.go"))
	require.NotContains(t, walk(t, fsys), "public")
	require.Contains(t, walk(t, fsys), "src/page.go")
}

func TestReadDirPagesVisibleEntries(t *testing.T) {
	tree := fstest.MapFS{
		".gitignore":   {Data: []byte("*.tmp\n")},
		"pkg/keep.go":  {Data: []byte("package p\n")},
		"pkg/drop.tmp": {Data: []byte("x")},
		"pkg/next.go":  {Data: []byte("package p\n")},
	}
	fsys := collect(t, tree)
	file, err := fsys.Open("pkg")
	require.NoError(t, err)
	directory := file.(fs.ReadDirFile)
	var names []string
	for {
		entries, err := directory.ReadDir(1)
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if len(entries) == 0 {
			break
		}
	}
	require.NoError(t, file.Close())
	require.ElementsMatch(t, []string{"keep.go", "next.go"}, names)
}

func TestGlobalExcludesAreNotLoaded(t *testing.T) {
	config := t.TempDir()
	excludes := config + "/exclude"
	require.NoError(t, os.WriteFile(excludes, []byte("*.fromglobal\n"), 0o644))
	gitconfig := config + "/gitconfig"
	require.NoError(t, os.WriteFile(gitconfig, []byte("[core]\n\texcludesfile = "+excludes+"\n"), 0o644))
	t.Setenv("GIT_CONFIG_GLOBAL", gitconfig)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	xdg := t.TempDir()
	require.NoError(t, os.MkdirAll(xdg+"/git", 0o755))
	require.NoError(t, os.WriteFile(xdg+"/git/ignore", []byte("*.fromglobal\n"), 0o644))
	t.Setenv("XDG_CONFIG_HOME", xdg)

	fsys := collect(t, fstest.MapFS{"note.fromglobal": {Data: []byte("x")}})
	require.False(t, fsys.Ignored("note.fromglobal"))
	require.Contains(t, walk(t, fsys), "note.fromglobal")
}

func TestNilFS(t *testing.T) {
	var fsys *ignore.FS
	require.False(t, fsys.Ignored("a"))
	require.False(t, fsys.SkipDir("a"))
}

type fileFS struct{}

func (fileFS) Open(name string) (fs.File, error) {
	return &fileOnly{name: name}, nil
}

type fileOnly struct{ name string }

func (f *fileOnly) Stat() (fs.FileInfo, error) { return fileInfo{name: f.name}, nil }
func (f *fileOnly) Read([]byte) (int, error)   { return 0, io.EOF }
func (f *fileOnly) Close() error               { return nil }

type fileInfo struct{ name string }

func (fileInfo) Name() string       { return "file" }
func (fileInfo) Size() int64        { return 0 }
func (fileInfo) Mode() fs.FileMode  { return 0o644 }
func (fileInfo) ModTime() time.Time { return time.Time{} }
func (fileInfo) IsDir() bool        { return false }
func (fileInfo) Sys() any           { return nil }

func collect(t *testing.T, tree fs.FS) *ignore.FS {
	t.Helper()
	fsys, err := ignore.Collect(t.Context(), tree, ignore.Config{})
	require.NoError(t, err)
	return fsys
}

func walk(t *testing.T, fsys fs.FS) []string {
	t.Helper()
	var names []string
	err := fs.WalkDir(fsys, ".", func(name string, _ fs.DirEntry, err error) error {
		require.NoError(t, err)
		names = append(names, name)
		return nil
	})
	require.NoError(t, err)
	return names
}

func dirNames(t *testing.T, fsys fs.FS, name string) []string {
	t.Helper()
	entries, err := fs.ReadDir(fsys, name)
	require.NoError(t, err)
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}
