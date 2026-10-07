package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/sops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const publisherAgeConfig = "creation_rules:\n  - age: age1zufvjtsk0p7wgsz7nth4032t4tqmev7d4dwq72x6cgngjqcnxgmq6l7ts0\n"

func TestKeyRunWritesWithoutPrompt(t *testing.T) {
	useSopsProgram(t)
	usePublisherAgeKey(t)
	t.Setenv("LEWKIT_SIGN_P12", "")
	dir := t.TempDir()
	writePublisherSops(t, dir)
	out := filepath.Join(dir, "keys", "publisher.p12")

	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme", "--out", out)
	require.NoError(t, app.Args.release.key.Run(t.Context()))

	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	id := openPublisherKey(t, out)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func TestKeyPromptsForMissingFields(t *testing.T) {
	useSopsProgram(t)
	usePublisherAgeKey(t)
	t.Setenv("LEWKIT_SIGN_P12", "")
	out := filepath.Join(t.TempDir(), "publisher.p12")
	writePublisherSops(t, filepath.Dir(out))
	var prompts []string
	ask := func(_ context.Context, prompt string) (string, error) {
		prompts = append(prompts, prompt)
		switch prompt {
		case "Publisher name":
			return "Acme", nil
		case "PKCS#12 path":
			return out, nil
		default:
			return "", assert.AnError
		}
	}
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key")
	require.NoError(t, app.Args.release.key.generate(t.Context(), ask, refuseConfirm))
	assert.Equal(t, []string{"Publisher name", "PKCS#12 path"}, prompts)
	id := openPublisherKey(t, out)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func TestKeyRejectsEmptyName(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--out", filepath.Join(t.TempDir(), "k.p12"))
	ask := func(context.Context, string) (string, error) { return "  ", nil }
	err := app.Args.release.key.generate(t.Context(), ask, refuseConfirm)
	require.ErrorIs(t, err, errNameRequired)
}

func TestKeyMissingSopsConfig(t *testing.T) {
	useSopsProgram(t)
	t.Setenv("LEWKIT_SIGN_P12", "")
	out := filepath.Join(t.TempDir(), "publisher.p12")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme", "--out", out)
	err := app.Args.release.key.generate(t.Context(), unexpectedAsk, refuseConfirm)
	require.ErrorIs(t, err, sops.ErrNoConfig)
	_, statErr := os.Stat(out)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestKeyConfirmsReplace(t *testing.T) {
	useSopsProgram(t)
	usePublisherAgeKey(t)
	t.Setenv("LEWKIT_SIGN_P12", "")
	dir := t.TempDir()
	writePublisherSops(t, dir)
	out := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o600))

	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme", "--out", out)
	err := app.Args.release.key.generate(t.Context(), unexpectedAsk, func(context.Context, string) (bool, error) {
		return false, nil
	})
	require.ErrorIs(t, err, errReplaceRefused)
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "old", string(raw))

	var asked string
	err = app.Args.release.key.generate(t.Context(), unexpectedAsk, func(_ context.Context, message string) (bool, error) {
		asked = message
		return true, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "Replace "+out+"?", asked)
	id := openPublisherKey(t, out)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func TestKeyForceReplaces(t *testing.T) {
	useSopsProgram(t)
	usePublisherAgeKey(t)
	t.Setenv("LEWKIT_SIGN_P12", "")
	dir := t.TempDir()
	writePublisherSops(t, dir)
	out := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o600))
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--force", "--name", "Acme", "--out", out)
	require.NoError(t, app.Args.release.key.generate(t.Context(), unexpectedAsk, refuseConfirm))
	id := openPublisherKey(t, out)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func useSopsProgram(t *testing.T) {
	t.Helper()
	dir := filepath.Join(os.TempDir(), "lewkit-sops-bin")
	bin := filepath.Join(dir, "sops")
	if _, err := os.Stat(bin); err != nil {
		require.NoError(t, os.MkdirAll(dir, 0o755))
		cmd := exec.Command("go", "build", "-o", bin, "github.com/getsops/sops/v3/cmd/sops")
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writePublisherSops(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(publisherAgeConfig), 0o644))
}

func usePublisherAgeKey(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, key := range []string{
		"SOPS_AGE_KEY_FILE",
		"SOPS_AGE_KEY_CMD",
		"SOPS_AGE_SSH_PRIVATE_KEY_FILE",
	} {
		unsetTestEnv(t, key)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "x", "sops", "testdata", "age.txt"))
	require.NoError(t, err)
	t.Setenv("SOPS_AGE_KEY", string(raw))
}

func unsetTestEnv(t *testing.T, key string) {
	t.Helper()
	prev, ok := os.LookupEnv(key)
	t.Cleanup(func() {
		if ok {
			os.Setenv(key, prev)
			return
		}
		os.Unsetenv(key)
	})
	os.Unsetenv(key)
}

func openPublisherKey(t *testing.T, path string) *sign.Identity {
	t.Helper()
	stored, err := os.ReadFile(path)
	require.NoError(t, err)
	plain, err := sops.Open(path)
	require.NoError(t, err)
	require.NotEqual(t, stored, plain)
	id, err := sign.LoadPKCS12(plain, "")
	require.NoError(t, err)
	return id
}

func unexpectedAsk(context.Context, string) (string, error) {
	return "", assert.AnError
}

func refuseConfirm(context.Context, string) (bool, error) {
	return false, assert.AnError
}
