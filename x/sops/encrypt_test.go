package sops

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/exec/native"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	dir := filepath.Join(os.TempDir(), "lewkit-sops-bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	bin := filepath.Join(dir, "sops")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/getsops/sops/v3/cmd/sops")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "build sops:", err)
		os.Exit(1)
	}
	os.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	os.Exit(m.Run())
}

const ageConfig = "creation_rules:\n  - age: age1zufvjtsk0p7wgsz7nth4032t4tqmev7d4dwq72x6cgngjqcnxgmq6l7ts0\n"

func TestEncryptRoundTrip(t *testing.T) {
	useAgeKey(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(ageConfig), 0o644))
	path := filepath.Join(dir, "keys", "publisher.p12")
	plain := []byte{0x30, 0x82, 0x01, 0x00, 0xff, 0x00, 'p', '1', '2'}
	out, err := Encrypt(t.Context(), path, plain)
	require.NoError(t, err)
	require.NotEqual(t, plain, out)
	require.True(t, bytes.HasPrefix(bytes.TrimSpace(out), []byte("{")))

	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, out, 0o600))
	got, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, plain, got)
}

func TestEncryptUsesParentConfig(t *testing.T) {
	useAgeKey(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(ageConfig), 0o644))
	path := filepath.Join(dir, "nested", "key.p12")
	plain := []byte("pkcs12")
	out, err := Encrypt(t.Context(), path, plain)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, out, 0o600))
	got, err := Open(path)
	require.NoError(t, err)
	require.Equal(t, plain, got)
}

func TestEncryptMissingConfig(t *testing.T) {
	_, err := Encrypt(t.Context(), filepath.Join(t.TempDir(), "key.p12"), []byte("plain"))
	require.ErrorIs(t, err, ErrNoConfig)
}

func TestEncryptRejectsSopsYml(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yml"), []byte(ageConfig), 0o644))
	_, err := Encrypt(t.Context(), filepath.Join(dir, "key.p12"), []byte("plain"))
	require.ErrorIs(t, err, ErrNoConfig)
	require.ErrorContains(t, err, ".sops.yaml")
}

func TestEncryptNoCreationRules(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte("{}\n"), 0o644))
	_, err := Encrypt(t.Context(), filepath.Join(dir, "key.p12"), []byte("plain"))
	require.ErrorIs(t, err, ErrNoConfig)
	require.ErrorContains(t, err, "no creation rules")
}

func TestEncryptUnmatchedRule(t *testing.T) {
	dir := t.TempDir()
	cfg := "creation_rules:\n  - path_regex: \\.env$\n    age: age1zufvjtsk0p7wgsz7nth4032t4tqmev7d4dwq72x6cgngjqcnxgmq6l7ts0\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".sops.yaml"), []byte(cfg), 0o644))
	_, err := Encrypt(t.Context(), filepath.Join(dir, "key.p12"), []byte("plain"))
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrNoConfig)
}
