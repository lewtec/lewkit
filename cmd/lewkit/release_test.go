package main

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoCompilerFlag(t *testing.T) {
	built := cmd.ParseOK[cmd.App[root]](t, "release", "build")
	assert.Equal(t, gocmd.CompilerLewkit, built.Args.release.build.compiler.Value())
	run := cmd.ParseOK[cmd.App[root]](t, "release", "run")
	assert.Equal(t, gocmd.CompilerLewkit, run.Args.release.run.compiler.Value())
	all := cmd.ParseOK[cmd.App[root]](t, "release", "build-all")
	assert.Equal(t, gocmd.CompilerLewkit, all.Args.release.buildAll.compiler.Value())

	host := cmd.ParseOK[cmd.App[root]](t, "release", "build", "--go", "host")
	assert.Equal(t, gocmd.CompilerHost, host.Args.release.build.compiler.Value())
	_, err := cmd.Parse[cmd.App[root]]("release", "build", "--go", "tinygo")
	assert.ErrorIs(t, err, cmd.ErrInvalidArgument)
}

func TestProgramGoOnly(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--go-only")
	fn, err := app.Args.release.run.program(t.Context(), []string{"a.tar.gz"})
	require.NoError(t, err)
	assert.Nil(t, fn)
}

func TestProgramEmptyPaths(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run")
	fn, err := app.Args.release.run.program(t.Context(), nil)
	require.NoError(t, err)
	assert.Nil(t, fn)
}

func TestProgramHostRejectsArgs(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--app", "--config", "eletrocromo.json", "--", "nope")
	_, err := app.Args.release.run.program(t.Context(), []string{"app.apk"})
	require.ErrorIs(t, err, errHostArgs)
}

func TestConfigDefaultsToEletrocromoJSON(t *testing.T) {
	run := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--app")
	assert.Equal(t, "./eletrocromo.json", run.Args.release.run.config.Value())
	assert.Equal(t, "./eletrocromo.json", run.Args.release.run.spec().Config)
	build := cmd.ParseOK[cmd.App[root]](t, "release", "build")
	assert.Equal(t, "./eletrocromo.json", build.Args.release.build.config.Value())
	assert.Equal(t, "./eletrocromo.json", build.Args.release.build.spec().Config)
}

func TestDirFlagRequiresConfigInThatDirectory(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "build", "--dir", "games")
	assert.True(t, app.Args.release.build.dir.ArgSet())
	assert.Equal(t, filepath.Join("games", "eletrocromo.json"), app.Args.release.build.spec().Config)
}

func TestOverlayFlagsOmitMissingDefaultConfig(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--entry", "static:city-racing.zip", "--id", "teste.cityracing")
	spec := app.Args.release.run.spec()
	assert.Empty(t, spec.Config)
	assert.Equal(t, "teste.cityracing", spec.ID)
	assert.Equal(t, "static:city-racing.zip", spec.Entry)

	explicit := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--config", "gone.json", "--entry", "static:city-racing.zip", "--id", "teste.cityracing")
	assert.Equal(t, "gone.json", explicit.Args.release.run.spec().Config)
}

func TestBlankFlagDoesNotCountAsOverlay(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--id", "   ", "--name", " ")
	assert.Equal(t, "./eletrocromo.json", app.Args.release.run.spec().Config)
}

func TestDirAndEntryOverlayTheFile(t *testing.T) {
	dir := t.TempDir()
	writeStoredZip(t, filepath.Join(dir, "city-racing.zip"), "<h1>city</h1>")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "City Racing",
		"entry": "go:."
	}`), 0o644))
	t.Chdir(t.TempDir())

	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--dir", dir, "--entry", "static:city-racing.zip", "--id", "teste.cityracing")
	spec := app.Args.release.run.spec()
	require.Empty(t, spec.Config)
	require.Equal(t, dir, spec.Dir)
	cfg, base, err := spec.Load(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
	require.Equal(t, dir, base)
	require.Equal(t, "teste.cityracing", cfg.PackageID)
	require.Equal(t, "City Racing", cfg.AppName)
	copied, err := os.ReadFile(filepath.Join(cfg.GoMain, "site.zip"))
	require.NoError(t, err)
	require.Contains(t, string(copied), "<h1>city</h1>")
}

func TestReleaseRunFlagsWithoutConfigFile(t *testing.T) {
	dir := t.TempDir()
	writeStoredZip(t, filepath.Join(dir, "city-racing.zip"), "<h1>city</h1>")
	t.Chdir(dir)

	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--entry", "static:city-racing.zip", "--id", "teste.cityracing", "--name", "City Racing")
	cfg, _, err := app.Args.release.run.spec().Load(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
	require.Equal(t, "teste.cityracing", cfg.PackageID)
	require.Equal(t, "City Racing", cfg.AppName)
	body, err := os.ReadFile(filepath.Join(cfg.GoMain, "main.go"))
	require.NoError(t, err)
	require.Contains(t, string(body), `"City Racing"`)
	copied, err := os.ReadFile(filepath.Join(cfg.GoMain, "site.zip"))
	require.NoError(t, err)
	require.Contains(t, string(copied), "<h1>city</h1>")
}

func TestP12FlagReadsSecretFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(keyPath, []byte{0x30, 0x82}, 0o600))

	app := cmd.ParseOK[cmd.App[root]](t, "release", "build", "--p12", keyPath)
	assert.Equal(t, []byte{0x30, 0x82}, app.Args.release.build.p12.Value())

	_, err := app.Args.release.build.identity()
	require.ErrorIs(t, err, sign.ErrPKCS12)
}

func TestP12EnvReadsSecretFile(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(keyPath, []byte("pkcs"), 0o600))
	t.Setenv("LEWKIT_SIGN_P12", keyPath)

	app := cmd.ParseOK[cmd.App[root]](t, "release", "build")
	assert.Equal(t, []byte("pkcs"), app.Args.release.build.p12.Value())
}

func TestIdentityUnset(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "build")
	id, err := app.Args.release.build.identity()
	require.NoError(t, err)
	assert.Nil(t, id)
}

func TestAppUsesDefaultConfig(t *testing.T) {
	app := cmd.ParseOK[cmd.App[root]](t, "release", "build", "--app", "--go-only")
	_, err := app.Args.release.build.produce(t.Context())
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestProgramStartsAfterProgress(t *testing.T) {
	if runtime.GOOS == "windows" {
		test.Skip(t, "shell script")
	}
	archive := scriptArchive(t, "#!/bin/sh\ntest \"$1\" = smoke\n")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--", "smoke")
	fn, err := app.Args.release.run.program(t.Context(), []string{archive})
	require.NoError(t, err)
	require.NotNil(t, fn)
	require.NoError(t, entry.Run(t.Context(), func(context.Context) error {
		entry.After(fn)
		return nil
	}))
}

func writeStoredZip(t *testing.T, path, body string) {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	header := &zip.FileHeader{Name: "index.html", Method: zip.Store}
	file, err := writer.CreateHeader(header)
	require.NoError(t, err)
	_, err = file.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
}

func TestRunNestedReturnsBuildError(t *testing.T) {
	session, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	app := cmd.ParseOK[cmd.App[root]](t, "release", "run", "--config", filepath.Join(t.TempDir(), "missing.json"))
	var runErr error
	err := progress.Run(session, ctx, func(ctx context.Context) error {
		runErr = app.Args.release.run.Run(ctx)
		return nil
	})
	require.Error(t, runErr)
	require.Error(t, err)
}
