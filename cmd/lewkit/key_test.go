package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/sops"
	"github.com/lewtec/lewkit/x/ui/tui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const publisherAgeConfig = "creation_rules:\n  - age: age1zufvjtsk0p7wgsz7nth4032t4tqmev7d4dwq72x6cgngjqcnxgmq6l7ts0\n"

func TestKeyRunWritesWithoutPrompt(t *testing.T) {
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
	usePublisherAgeKey(t)
	t.Setenv("LEWKIT_SIGN_P12", "")
	out := filepath.Join(t.TempDir(), "publisher.p12")
	writePublisherSops(t, filepath.Dir(out))
	var prompts []string
	ask := func(_ context.Context, questions []tui.Question) ([]tui.Answer, error) {
		answers := make([]tui.Answer, len(questions))
		for i, q := range questions {
			prompts = append(prompts, q.Prompt)
			switch q.Prompt {
			case "Publisher name":
				answers[i] = tui.Answer{Text: "Acme"}
			case "PKCS#12 path":
				answers[i] = tui.Answer{Text: out}
			default:
				return nil, assert.AnError
			}
		}
		return answers, nil
	}
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key")
	require.NoError(t, app.Args.release.key.generate(t.Context(), ask))
	assert.Equal(t, []string{"Publisher name", "PKCS#12 path"}, prompts)
	id := openPublisherKey(t, out)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func TestKeyRejectsEmptyName(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--out", filepath.Join(t.TempDir(), "k.p12"))
	ask := func(context.Context, []tui.Question) ([]tui.Answer, error) {
		return []tui.Answer{{Text: "  "}}, nil
	}
	err := app.Args.release.key.generate(t.Context(), ask)
	require.ErrorIs(t, err, errNameRequired)
}

func TestKeyRejectsDirectory(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme", "--out", t.TempDir())
	err := app.Args.release.key.generate(t.Context(), refuseInterview)
	require.ErrorIs(t, err, errPathIsDir)
}

func TestKeyInterviewCancelWritesNothing(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	err := cmd.ParseOK[cmd.App[root]](t, "release", "key").Args.release.key.generate(t.Context(), func(context.Context, []tui.Question) ([]tui.Answer, error) {
		return nil, tui.ErrCanceled
	})
	require.ErrorIs(t, err, tui.ErrCanceled)
}

func TestKeyMissingSopsConfig(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	out := filepath.Join(t.TempDir(), "publisher.p12")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme", "--out", out)
	err := app.Args.release.key.generate(t.Context(), refuseInterview)
	require.ErrorIs(t, err, sops.ErrNoConfig)
	_, statErr := os.Stat(out)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestKeyConfirmsReplace(t *testing.T) {
	usePublisherAgeKey(t)
	t.Setenv("LEWKIT_SIGN_P12", "")
	dir := t.TempDir()
	writePublisherSops(t, dir)
	out := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o600))

	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme", "--out", out)
	err := app.Args.release.key.generate(t.Context(), func(_ context.Context, questions []tui.Question) ([]tui.Answer, error) {
		require.Len(t, questions, 1)
		assert.True(t, questions[0].Confirm)
		assert.Equal(t, "Replace "+out+"?", questions[0].Prompt)
		return []tui.Answer{{Text: "n"}}, nil
	})
	require.ErrorIs(t, err, errReplaceRefused)
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "old", string(raw))

	err = app.Args.release.key.generate(t.Context(), func(_ context.Context, questions []tui.Question) ([]tui.Answer, error) {
		require.Equal(t, []tui.Question{{Prompt: "Replace " + out + "?", Confirm: true}}, questions)
		return []tui.Answer{{Yes: true, Text: "y"}}, nil
	})
	require.NoError(t, err)
	id := openPublisherKey(t, out)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func TestKeyConfirmsPromptedReplace(t *testing.T) {
	usePublisherAgeKey(t)
	t.Setenv("LEWKIT_SIGN_P12", "")
	dir := t.TempDir()
	writePublisherSops(t, dir)
	out := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o600))

	var prompts []string
	ask := func(_ context.Context, questions []tui.Question) ([]tui.Answer, error) {
		answers := make([]tui.Answer, len(questions))
		for i, q := range questions {
			prompts = append(prompts, q.Prompt)
			if q.Confirm {
				answers[i] = tui.Answer{Yes: true, Text: "y"}
				continue
			}
			answers[i] = tui.Answer{Text: out}
		}
		return answers, nil
	}
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme")
	require.NoError(t, app.Args.release.key.generate(t.Context(), ask))
	assert.Equal(t, []string{"PKCS#12 path", "Replace " + out + "?"}, prompts)
	id := openPublisherKey(t, out)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func TestKeyForceReplaces(t *testing.T) {
	usePublisherAgeKey(t)
	t.Setenv("LEWKIT_SIGN_P12", "")
	dir := t.TempDir()
	writePublisherSops(t, dir)
	out := filepath.Join(dir, "publisher.p12")
	require.NoError(t, os.WriteFile(out, []byte("old"), 0o600))
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--force", "--name", "Acme", "--out", out)
	require.NoError(t, app.Args.release.key.generate(t.Context(), refuseInterview))
	id := openPublisherKey(t, out)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
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

func refuseInterview(context.Context, []tui.Question) ([]tui.Answer, error) {
	return nil, assert.AnError
}
