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
	ask := func(_ context.Context, questions []tui.Question, _ tui.Grow) ([]tui.Answer, error) {
		answers := make([]tui.Answer, len(questions))
		for i, q := range questions {
			prompts = append(prompts, q.Prompt)
			switch q.Prompt {
			case "certificate common name":
				answers[i] = tui.Answer{Text: "Acme"}
			case "PKCS#12 file to write":
				answers[i] = tui.Answer{Text: out}
			default:
				return nil, assert.AnError
			}
		}
		return answers, nil
	}
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key")
	require.NoError(t, app.Args.release.key.generate(t.Context(), ask))
	assert.Equal(t, []string{"certificate common name", "PKCS#12 file to write"}, prompts)
	id := openPublisherKey(t, out)
	assert.Equal(t, "Acme", id.Certs[0].Subject.CommonName)
}

func TestKeyOutDefaultsToPublisherFile(t *testing.T) {
	unsetTestEnv(t, "LEWKIT_SIGN_P12")
	key := cmd.ParseOK[cmd.App[root]](t, "release", "key").Args.release.key
	assert.False(t, key.out.ArgSet())
	assert.Equal(t, "publisher.p12", key.out.Value())
	asks, err := cmd.Asks(key)
	require.NoError(t, err)
	var path cmd.Ask
	var saw bool
	for _, ask := range asks {
		if ask.Name == "out" {
			path = ask
			saw = true
		}
	}
	require.True(t, saw)
	assert.Equal(t, "publisher.p12", path.Default)
	assert.Equal(t, "PKCS#12 file to write", path.Prompt)
}

func TestKeyRejectsEmptyName(t *testing.T) {
	t.Setenv("LEWKIT_SIGN_P12", "")
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--out", filepath.Join(t.TempDir(), "k.p12"))
	ask := func(context.Context, []tui.Question, tui.Grow) ([]tui.Answer, error) {
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
	err := cmd.ParseOK[cmd.App[root]](t, "release", "key").Args.release.key.generate(t.Context(), func(context.Context, []tui.Question, tui.Grow) ([]tui.Answer, error) {
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
	err := app.Args.release.key.generate(t.Context(), func(_ context.Context, questions []tui.Question, grow tui.Grow) ([]tui.Answer, error) {
		require.Nil(t, grow)
		require.Equal(t, []tui.Question{replaceQuestion(out)}, questions)
		return []tui.Answer{{Text: "no"}}, nil
	})
	require.ErrorIs(t, err, errReplaceRefused)
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Equal(t, "old", string(raw))

	err = app.Args.release.key.generate(t.Context(), func(_ context.Context, questions []tui.Question, _ tui.Grow) ([]tui.Answer, error) {
		require.Equal(t, []tui.Question{replaceQuestion(out)}, questions)
		return []tui.Answer{{Text: "yes", Yes: true}}, nil
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
	ask := func(_ context.Context, questions []tui.Question, grow tui.Grow) ([]tui.Answer, error) {
		answers := make([]tui.Answer, 0, len(questions)+1)
		for _, q := range questions {
			prompts = append(prompts, q.Prompt)
			if len(q.Choices) > 0 {
				return nil, assert.AnError
			}
			answers = append(answers, tui.Answer{Text: out})
		}
		require.NotNil(t, grow)
		extra, err := grow(answers)
		require.NoError(t, err)
		for _, q := range extra {
			prompts = append(prompts, q.Prompt)
			answers = append(answers, tui.Answer{Text: "yes", Yes: true})
		}
		return answers, nil
	}
	app := cmd.ParseOK[cmd.App[root]](t, "release", "key", "--name", "Acme")
	require.NoError(t, app.Args.release.key.generate(t.Context(), ask))
	assert.Equal(t, []string{"PKCS#12 file to write", "Replace " + out + "?"}, prompts)
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

func refuseInterview(context.Context, []tui.Question, tui.Grow) ([]tui.Answer, error) {
	return nil, assert.AnError
}
