package applications

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/tool"

	"github.com/stretchr/testify/require"
)

func TestFixRubyShebangs(t *testing.T) {
	directory := t.TempDir()
	binDirectory := filepath.Join(directory, "bin")
	require.NoError(t, os.MkdirAll(binDirectory, 0o755))

	rubyBinary := filepath.Join(binDirectory, "ruby")
	require.NoError(t, os.WriteFile(rubyBinary, []byte("ELF..."), 0o755))

	badScript := filepath.Join(binDirectory, "bundle")
	require.NoError(t, os.WriteFile(badScript, []byte("#!/opt/hostedtoolcache/Ruby/4.0.5/x64/bin/ruby\nputs 'hello'\n"), 0o755))

	badWithArgs := filepath.Join(binDirectory, "rake")
	require.NoError(t, os.WriteFile(badWithArgs, []byte("#!/opt/hostedtoolcache/Ruby/4.0.5/x64/bin/ruby -w\nputs 'rake'\n"), 0o755))

	good := filepath.Join(binDirectory, "good")
	goodContent := "#!" + rubyBinary + "\nputs 'ok'\n"
	require.NoError(t, os.WriteFile(good, []byte(goodContent), 0o755))

	other := filepath.Join(binDirectory, "other")
	require.NoError(t, os.WriteFile(other, []byte("#!/bin/sh\necho hi\n"), 0o755))

	require.NoError(t, (&rubyTool{}).fixRubyShebangs(t.Context(), directory))

	got, err := os.ReadFile(badScript)
	require.NoError(t, err)
	require.Equal(t, "#!"+rubyBinary+"\nputs 'hello'\n", string(got))

	got, err = os.ReadFile(badWithArgs)
	require.NoError(t, err)
	require.Equal(t, "#!"+rubyBinary+" -w\nputs 'rake'\n", string(got))

	got, err = os.ReadFile(good)
	require.NoError(t, err)
	require.Equal(t, goodContent, string(got))

	got, err = os.ReadFile(other)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(got, []byte("#!/bin/sh")))
}

func TestRubyToolImplementsInstallFixer(t *testing.T) {
	var _ tool.Fixer = (*rubyTool)(nil)
	installed := &rubyTool{}

	directory := t.TempDir()
	binDirectory := filepath.Join(directory, "bin")
	require.NoError(t, os.MkdirAll(binDirectory, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDirectory, "ruby"), []byte("fake"), 0o755))
	script := filepath.Join(binDirectory, "irb")
	require.NoError(t, os.WriteFile(script, []byte("#!/opt/hostedtoolcache/Ruby/4.0.5/x64/bin/ruby\n# gem wrapper\n"), 0o755))

	require.NoError(t, installed.Fix(t.Context(), directory))
	body, err := os.ReadFile(script)
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(body, []byte("#!"+filepath.Join(directory, "bin", "ruby"))))
}
