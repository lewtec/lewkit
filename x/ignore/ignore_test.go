package ignore_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/ignore"
)

func TestCollectEmptyAndFileRoot(t *testing.T) {
	_, err := ignore.Collect(t.Context(), "", ignore.Config{})
	require.ErrorIs(t, err, lewpath.ErrEmptyPath)

	dir := t.TempDir()
	file := filepath.Join(dir, "note.txt")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	_, err = ignore.Collect(t.Context(), file, ignore.Config{})
	require.ErrorIs(t, err, lewpath.ErrNotDir)
}

func TestCollectCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := ignore.Collect(ctx, t.TempDir(), ignore.Config{})
	require.ErrorIs(t, err, context.Canceled)
}

func TestGitignoreNestedNegation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "*.tmp\n")
	writeFile(t, filepath.Join(root, "pkg", ".gitignore"), "!keep.tmp\n")
	writeFile(t, filepath.Join(root, "a.tmp"), "x")
	writeFile(t, filepath.Join(root, "pkg", "keep.tmp"), "x")
	writeFile(t, filepath.Join(root, "pkg", "drop.tmp"), "x")

	set := collect(t, root)
	require.True(t, set.Ignored(filepath.Join(root, "a.tmp")))
	require.False(t, set.Ignored(filepath.Join(root, "pkg", "keep.tmp")))
	require.True(t, set.Ignored(filepath.Join(root, "pkg", "drop.tmp")))
	require.False(t, set.SkipDir(filepath.Join(root, "pkg")))
}

func TestParentDirectoryRuleIsNotUndone(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "build/\n!build/keep.go\n")
	writeFile(t, filepath.Join(root, "build", "a.go"), "package b\n")
	writeFile(t, filepath.Join(root, "build", "keep.go"), "package b\n")

	set := collect(t, root)
	require.True(t, set.Ignored(filepath.Join(root, "build")))
	require.True(t, set.Ignored(filepath.Join(root, "build", "a.go")))
	require.True(t, set.Ignored(filepath.Join(root, "build", "keep.go")))
	require.True(t, set.SkipDir(filepath.Join(root, "build")))
}

func TestContentsNegationStillApplies(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "build/*\n!build/keep.go\n")
	writeFile(t, filepath.Join(root, "build", "a.go"), "package b\n")
	writeFile(t, filepath.Join(root, "build", "keep.go"), "package b\n")

	set := collect(t, root)
	require.False(t, set.Ignored(filepath.Join(root, "build")))
	require.True(t, set.Ignored(filepath.Join(root, "build", "a.go")))
	require.False(t, set.Ignored(filepath.Join(root, "build", "keep.go")))
	require.False(t, set.SkipDir(filepath.Join(root, "build")))
}

func TestStarStarAllowsNegationInside(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "gen/**\n!gen/keep.go\n")
	writeFile(t, filepath.Join(root, "gen", "a.go"), "package g\n")
	writeFile(t, filepath.Join(root, "gen", "keep.go"), "package g\n")

	set := collect(t, root)
	require.True(t, set.Ignored(filepath.Join(root, "gen", "a.go")))
	require.False(t, set.Ignored(filepath.Join(root, "gen", "keep.go")))
	require.False(t, set.SkipDir(filepath.Join(root, "gen")))

	closed := t.TempDir()
	writeFile(t, filepath.Join(closed, ".gitignore"), "gen/**\n")
	writeFile(t, filepath.Join(closed, "gen", "a.go"), "package g\n")
	closedSet := collect(t, closed)
	require.True(t, closedSet.Ignored(filepath.Join(closed, "gen", "a.go")))
	require.True(t, closedSet.SkipDir(filepath.Join(closed, "gen")))
}

func TestExcludedDirectoryBlocksChildNegation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "*\n!keep.go\n")
	writeFile(t, filepath.Join(root, "keep.go"), "package k\n")
	writeFile(t, filepath.Join(root, "sub", "keep.go"), "package k\n")
	writeFile(t, filepath.Join(root, "sub", "other.go"), "package k\n")

	set := collect(t, root)
	require.False(t, set.Ignored(filepath.Join(root, "keep.go")))
	require.True(t, set.Ignored(filepath.Join(root, "sub", "keep.go")))
	require.True(t, set.Ignored(filepath.Join(root, "sub", "other.go")))
	require.True(t, set.SkipDir(filepath.Join(root, "sub")))
}

func TestReincludeDirectoryThenFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "/*\n!/keep/\n/keep/*\n!/keep/yes.go\n")
	writeFile(t, filepath.Join(root, "keep", "yes.go"), "package k\n")
	writeFile(t, filepath.Join(root, "keep", "no.go"), "package k\n")

	set := collect(t, root)
	require.False(t, set.Ignored(filepath.Join(root, "keep")))
	require.False(t, set.Ignored(filepath.Join(root, "keep", "yes.go")))
	require.True(t, set.Ignored(filepath.Join(root, "keep", "no.go")))
	require.False(t, set.SkipDir(filepath.Join(root, "keep")))
}

func TestDeeperGitignoreWins(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "*.tmp\n")
	writeFile(t, filepath.Join(root, "a", ".gitignore"), "!*.tmp\n")
	writeFile(t, filepath.Join(root, "a", "b", ".gitignore"), "*.tmp\n")
	writeFile(t, filepath.Join(root, "a", "y.tmp"), "x")
	writeFile(t, filepath.Join(root, "a", "b", "x.tmp"), "x")

	set := collect(t, root)
	require.False(t, set.Ignored(filepath.Join(root, "a", "y.tmp")))
	require.True(t, set.Ignored(filepath.Join(root, "a", "b", "x.tmp")))
}

func TestExcludeThenGitignore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".git", "info", "exclude"), "secret.txt\n*.log\n")
	writeFile(t, filepath.Join(root, ".gitignore"), "!secret.txt\n")
	writeFile(t, filepath.Join(root, "secret.txt"), "x")
	writeFile(t, filepath.Join(root, "debug.log"), "x")

	set := collect(t, root)
	require.Equal(t, root, set.Root)
	require.False(t, set.Ignored(filepath.Join(root, "secret.txt")))
	require.True(t, set.Ignored(filepath.Join(root, "debug.log")))
}

func TestParentGitignoreAndExcludeFromToplevel(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".git", "info", "exclude"), "hidden.txt\n")
	writeFile(t, filepath.Join(root, ".gitignore"), "*.log\n")
	writeFile(t, filepath.Join(root, "svc", ".gitignore"), "!keep.log\n")
	writeFile(t, filepath.Join(root, "svc", "keep.log"), "x")
	writeFile(t, filepath.Join(root, "svc", "drop.log"), "x")
	writeFile(t, filepath.Join(root, "svc", "hidden.txt"), "x")
	writeFile(t, filepath.Join(root, "outside.log"), "x")

	set := collect(t, filepath.Join(root, "svc"))
	require.Equal(t, root, set.Root)
	require.False(t, set.Ignored(filepath.Join(root, "svc", "keep.log")))
	require.True(t, set.Ignored(filepath.Join(root, "svc", "drop.log")))
	require.True(t, set.Ignored(filepath.Join(root, "svc", "hidden.txt")))
	require.True(t, set.Ignored(filepath.Join(root, "outside.log")))
}

func TestOutsideCheckoutIsNotAParent(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "repo")
	writeFile(t, filepath.Join(outer, ".gitignore"), "*.nope\n")
	writeFile(t, filepath.Join(root, ".git", "HEAD"), "ref: refs/heads/main\n")
	writeFile(t, filepath.Join(root, "file.nope"), "x")

	set := collect(t, root)
	require.Equal(t, root, set.Root)
	require.False(t, set.Ignored(filepath.Join(root, "file.nope")))
}

func TestNoGitDoesNotWalkAboveRoot(t *testing.T) {
	outer := t.TempDir()
	root := filepath.Join(outer, "proj")
	writeFile(t, filepath.Join(outer, ".gitignore"), "*.nope\n")
	writeFile(t, filepath.Join(root, "file.nope"), "x")
	writeFile(t, filepath.Join(root, ".gitignore"), "*.tmp\n")
	writeFile(t, filepath.Join(root, "a.tmp"), "x")

	set := collect(t, root)
	require.Equal(t, root, set.Root)
	require.False(t, set.Ignored(filepath.Join(root, "file.nope")))
	require.True(t, set.Ignored(filepath.Join(root, "a.tmp")))
}

func TestWorktreeExclude(t *testing.T) {
	repo := t.TempDir()
	checkout := t.TempDir()
	writeFile(t, filepath.Join(repo, ".git", "info", "exclude"), "from-common.txt\n")
	writeFile(t, filepath.Join(repo, ".git", "worktrees", "wt", "commondir"), "../..\n")
	gitDir := filepath.Join(repo, ".git", "worktrees", "wt")
	writeFile(t, filepath.Join(checkout, ".git"), "gitdir: "+gitDir+"\n")
	writeFile(t, filepath.Join(checkout, "from-common.txt"), "x")
	writeFile(t, filepath.Join(checkout, "keep.txt"), "x")

	set := collect(t, checkout)
	require.Equal(t, checkout, set.Root)
	require.True(t, set.Ignored(filepath.Join(checkout, "from-common.txt")))
	require.False(t, set.Ignored(filepath.Join(checkout, "keep.txt")))
}

func TestSkipDirsAreOptional(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "node_modules", "x", "y.go"), "package x\n")
	writeFile(t, filepath.Join(root, "pkg", "y.go"), "package p\n")

	plain := collect(t, root)
	require.False(t, plain.Ignored(filepath.Join(root, "node_modules", "x", "y.go")))
	require.False(t, plain.SkipDir(filepath.Join(root, "node_modules")))

	set, err := ignore.Collect(t.Context(), root, ignore.Config{SkipDirs: []string{"node_modules", "node_modules", ""}})
	require.NoError(t, err)
	require.True(t, set.Ignored(filepath.Join(root, "node_modules", "x", "y.go")))
	require.True(t, set.SkipDir(filepath.Join(root, "node_modules")))
	require.False(t, set.Ignored(filepath.Join(root, "pkg", "y.go")))
	require.False(t, set.SkipDir(filepath.Join(root, "pkg")))
}

func TestSkipDirsStayInsideCollectedRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	writeFile(t, filepath.Join(root, "vendor", "a.go"), "package a\n")
	writeFile(t, filepath.Join(root, "svc", "vendor", "b.go"), "package b\n")
	writeFile(t, filepath.Join(root, "svc", "main.go"), "package svc\n")

	set, err := ignore.Collect(t.Context(), filepath.Join(root, "svc"), ignore.Config{SkipDirs: []string{"vendor"}})
	require.NoError(t, err)
	require.False(t, set.Ignored(filepath.Join(root, "vendor", "a.go")))
	require.True(t, set.Ignored(filepath.Join(root, "svc", "vendor", "b.go")))
	require.False(t, set.Ignored(filepath.Join(root, "svc", "main.go")))
}

func TestAttributesAreOptIn(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitattributes"), ""+
		"*.pb.go linguist-generated=true\n"+
		"gen/** linguist-generated\n"+
		"gen/keep.go -linguist-generated\n"+
		"*.snap refactree-ignored\n"+
		"hand.go linguist-generated=false\n"+
		"*.go linguist-language=Go\n")
	writeFile(t, filepath.Join(root, "api", "foo.pb.go"), "package api\n")
	writeFile(t, filepath.Join(root, "api", "hand.go"), "package api\n")
	writeFile(t, filepath.Join(root, "gen", "a.go"), "package gen\n")
	writeFile(t, filepath.Join(root, "gen", "keep.go"), "package gen\n")
	writeFile(t, filepath.Join(root, "x.snap"), "snap")
	writeFile(t, filepath.Join(root, "hand.go"), "package main\n")

	plain := collect(t, root)
	require.False(t, plain.Ignored(filepath.Join(root, "api", "foo.pb.go")))
	require.False(t, plain.Ignored(filepath.Join(root, "x.snap")))

	set, err := ignore.Collect(t.Context(), root, ignore.Config{Attributes: true})
	require.NoError(t, err)
	require.True(t, set.Ignored(filepath.Join(root, "api", "foo.pb.go")))
	require.False(t, set.Ignored(filepath.Join(root, "api", "hand.go")))
	require.True(t, set.Ignored(filepath.Join(root, "gen", "a.go")))
	require.False(t, set.Ignored(filepath.Join(root, "gen", "keep.go")))
	require.False(t, set.SkipDir(filepath.Join(root, "gen")))
	require.True(t, set.Ignored(filepath.Join(root, "x.snap")))
	require.False(t, set.Ignored(filepath.Join(root, "hand.go")))
}

func TestAttributeDirectoryRuleIsNotUndone(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitattributes"), ""+
		"dist/ linguist-generated\n"+
		"dist/keep.go -linguist-generated\n")
	writeFile(t, filepath.Join(root, "dist", "a.go"), "package d\n")
	writeFile(t, filepath.Join(root, "dist", "keep.go"), "package d\n")

	set, err := ignore.Collect(t.Context(), root, ignore.Config{Attributes: true})
	require.NoError(t, err)
	require.True(t, set.Ignored(filepath.Join(root, "dist", "a.go")))
	require.True(t, set.Ignored(filepath.Join(root, "dist", "keep.go")))
	require.True(t, set.SkipDir(filepath.Join(root, "dist")))
}

func TestNestedAttributes(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "pkg", ".gitattributes"), "*.gen.go linguist-generated\n")
	writeFile(t, filepath.Join(root, "pkg", "a.gen.go"), "package p\n")
	writeFile(t, filepath.Join(root, "pkg", "a.go"), "package p\n")

	set, err := ignore.Collect(t.Context(), root, ignore.Config{Attributes: true})
	require.NoError(t, err)
	require.True(t, set.Ignored(filepath.Join(root, "pkg", "a.gen.go")))
	require.False(t, set.Ignored(filepath.Join(root, "pkg", "a.go")))
}

func TestAnchoredDirectoryAndComments(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "# *.go\n\n/public/\n")
	writeFile(t, filepath.Join(root, "public", "index.html"), "<html>")
	writeFile(t, filepath.Join(root, "public", "en", "index.html"), "<html>")
	writeFile(t, filepath.Join(root, "src", "page.go"), "package src\n")

	set := collect(t, root)
	require.True(t, set.Ignored(filepath.Join(root, "public", "index.html")))
	require.True(t, set.Ignored(filepath.Join(root, "public", "en", "index.html")))
	require.True(t, set.SkipDir(filepath.Join(root, "public")))
	require.False(t, set.Ignored(filepath.Join(root, "src", "page.go")))
	require.False(t, set.Ignored(filepath.Join(root, "missing.go")))
}

func TestGlobalExcludesAreNotLoaded(t *testing.T) {
	config := filepath.Join(t.TempDir(), "gitconfig")
	excludes := filepath.Join(t.TempDir(), "exclude")
	writeFile(t, excludes, "*.fromglobal\n")
	writeFile(t, config, "[core]\n\texcludesfile = "+excludes+"\n")
	t.Setenv("GIT_CONFIG_GLOBAL", config)
	t.Setenv("GIT_CONFIG_SYSTEM", "/dev/null")
	xdg := t.TempDir()
	writeFile(t, filepath.Join(xdg, "git", "ignore"), "*.fromglobal\n")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "note.fromglobal"), "x")
	set := collect(t, root)
	require.False(t, set.Ignored(filepath.Join(root, "note.fromglobal")))
}

func TestNilSet(t *testing.T) {
	var set *ignore.Set
	require.False(t, set.Ignored("a"))
	require.False(t, set.SkipDir("a"))
}

func TestPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	writeFile(t, filepath.Join(root, ".gitignore"), "*\n")
	writeFile(t, filepath.Join(other, "a.txt"), "x")
	set := collect(t, root)
	require.False(t, set.Ignored(filepath.Join(other, "a.txt")))
	require.False(t, set.SkipDir(other))
}

func collect(t *testing.T, root string) *ignore.Set {
	t.Helper()
	set, err := ignore.Collect(t.Context(), root, ignore.Config{})
	require.NoError(t, err)
	return set
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o644))
}
