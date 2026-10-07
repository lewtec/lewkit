package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyRunWritesWithoutPrompt(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	t.Setenv("LEWKIT_SIGN_P12_PASSWORD", "")
	dir := t.TempDir()
	out := filepath.Join(dir, "keys", "publisher.p12")
	pass := filepath.Join(dir, "password.txt")
	require.NoError(t, os.WriteFile(pass, []byte("hunter2\n"), 0o600))

	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme", "--out", out, "--p12-password", pass)
	require.NoError(t, app.Args.release.key.Run(t.Context()))

	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	info, err := os.Stat(out)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	id, err := sign.LoadPKCS12(raw, "hunter2")
	require.NoError(t, err)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func TestKeyPromptsForMissingFields(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	t.Setenv("LEWKIT_SIGN_P12_PASSWORD", "")
	out := filepath.Join(t.TempDir(), "publisher.p12")
	var prompts []string
	ask := func(_ context.Context, prompt string) (string, error) {
		prompts = append(prompts, prompt)
		switch prompt {
		case "Publisher name":
			return "Acme", nil
		case "PKCS#12 path":
			return out, nil
		case "PKCS#12 password":
			return "hunter2", nil
		case "Repeat PKCS#12 password":
			return "hunter2", nil
		default:
			return "", assert.AnError
		}
	}
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key")
	require.NoError(t, app.Args.release.key.generate(t.Context(), ask, refuseConfirm))
	assert.Equal(t, []string{"Publisher name", "PKCS#12 path", "PKCS#12 password", "Repeat PKCS#12 password"}, prompts)
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	id, err := sign.LoadPKCS12(raw, "hunter2")
	require.NoError(t, err)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func TestKeyRejectsEmptyName(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	t.Setenv("LEWKIT_SIGN_P12_PASSWORD", "")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--out", filepath.Join(t.TempDir(), "k.p12"))
	ask := func(context.Context, string) (string, error) { return "  ", nil }
	err := app.Args.release.key.generate(t.Context(), ask, refuseConfirm)
	require.ErrorIs(t, err, errNameRequired)
}

func TestKeyRejectsPasswordMismatch(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	t.Setenv("LEWKIT_SIGN_P12_PASSWORD", "")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme", "--out", filepath.Join(t.TempDir(), "k.p12"))
	answers := []string{"one", "two"}
	ask := func(context.Context, string) (string, error) {
		next := answers[0]
		answers = answers[1:]
		return next, nil
	}
	err := app.Args.release.key.generate(t.Context(), ask, refuseConfirm)
	require.ErrorIs(t, err, errPasswordMismatch)
}

func TestKeyConfirmsReplace(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	t.Setenv("LEWKIT_SIGN_P12_PASSWORD", "")
	dir := t.TempDir()
	out := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o600))
	pass := filepath.Join(dir, "password.txt")
	require.NoError(t, os.WriteFile(pass, []byte("hunter2\n"), 0o600))

	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme", "--out", out, "--p12-password", pass)
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
	raw, err = os.ReadFile(out)
	require.NoError(t, err)
	_, err = sign.LoadPKCS12(raw, "hunter2")
	require.NoError(t, err)
}

func TestKeyForceReplaces(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	t.Setenv("LEWKIT_SIGN_P12_PASSWORD", "")
	dir := t.TempDir()
	out := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o600))
	pass := filepath.Join(dir, "password.txt")
	require.NoError(t, os.WriteFile(pass, []byte("hunter2\n"), 0o600))
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--force", "--name", "Acme", "--out", out, "--p12-password", pass)
	require.NoError(t, app.Args.release.key.generate(t.Context(), unexpectedAsk, refuseConfirm))
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	_, err = sign.LoadPKCS12(raw, "hunter2")
	require.NoError(t, err)
}

func unexpectedAsk(context.Context, string) (string, error) {
	return "", assert.AnError
}

func refuseConfirm(context.Context, string) (bool, error) {
	return false, assert.AnError
}
