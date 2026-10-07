package sops

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"filippo.io/age"
	"github.com/stretchr/testify/require"
)

func TestFileParseEmpty(t *testing.T) {
	var f File
	require.NoError(t, f.Parse("  "))
	require.Nil(t, f.Value())
	require.Empty(t, f.Text())
}

func TestPlainFilePassesThrough(t *testing.T) {
	raw := []byte{0x30, 0x82, 0xff, 0x00, 'p', '1', '2'}
	got, err := Decode(raw)
	require.NoError(t, err)
	require.Equal(t, raw, got)

	dir := t.TempDir()
	path := filepath.Join(dir, "password.txt")
	require.NoError(t, os.WriteFile(path, []byte("hunter2\n"), 0o600))
	var f File
	require.NoError(t, f.Parse(path))
	require.Equal(t, []byte("hunter2\n"), f.Value())
	require.Equal(t, "hunter2", f.Text())
}

func TestDecryptFixtures(t *testing.T) {
	useAgeKey(t)
	for _, tc := range []struct {
		enc   string
		plain string
	}{
		{"testdata/blob.sops", "testdata/blob.plain"},
		{"testdata/doc.json", "testdata/doc.plain"},
		{"testdata/secrets.yaml", "testdata/secrets.plain"},
		{"testdata/only.yaml", "testdata/only.plain"},
		{"testdata/env.env", "testdata/env.plain"},
	} {
		t.Run(tc.enc, func(t *testing.T) {
			want, err := os.ReadFile(tc.plain)
			require.NoError(t, err)
			got, err := Open(tc.enc)
			require.NoError(t, err)
			require.Equal(t, want, got)
		})
	}
}

func TestAgeKeyFile(t *testing.T) {
	key, err := os.ReadFile("testdata/age.txt")
	require.NoError(t, err)
	isolateAge(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "sops", "age", "keys.txt")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, append([]byte("# fixture\n"), key...), 0o600))
	t.Setenv("XDG_CONFIG_HOME", dir)

	want, err := os.ReadFile("testdata/secrets.plain")
	require.NoError(t, err)
	got, err := Open("testdata/secrets.yaml")
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestWrongAgeKey(t *testing.T) {
	isolateAge(t)
	id, err := age.GenerateX25519Identity()
	require.NoError(t, err)
	t.Setenv("SOPS_AGE_KEY", id.String())
	_, err = Open("testdata/blob.sops")
	require.ErrorIs(t, err, ErrNotAge)
	var chain interface{ Unwrap() []error }
	require.ErrorAs(t, err, &chain)
	require.GreaterOrEqual(t, len(chain.Unwrap()), 2)
}

func TestTamperFailsMAC(t *testing.T) {
	useAgeKey(t)
	raw, err := os.ReadFile("testdata/secrets.yaml")
	require.NoError(t, err)
	raw = bytes.Replace(raw, []byte("visible"), []byte("changed"), 1)
	_, err = Decode(raw)
	require.ErrorIs(t, err, ErrMAC)
}

func TestNotAge(t *testing.T) {
	const doc = `
name: x
sops:
    pgp:
        - fp: ABC
          enc: nope
          created_at: "2020-01-02T03:04:05Z"
    lastmodified: "2020-01-02T03:04:05Z"
    mac: ENC[AES256_GCM,data:aa,iv:bb,tag:cc,type:str]
    version: "3.10.2"
`
	_, err := Decode([]byte(doc))
	require.ErrorIs(t, err, ErrNotAge)
}

func TestShamirRejected(t *testing.T) {
	const doc = `
name: x
sops:
    shamir_threshold: 1
    key_groups:
        - age:
            - recipient: age1example
              enc: nope
        - age:
            - recipient: age1other
              enc: nope
    lastmodified: "2020-01-02T03:04:05Z"
    mac: ENC[AES256_GCM,data:aa,iv:bb,tag:cc,type:str]
`
	_, err := Decode([]byte(doc))
	require.Error(t, err)
	require.ErrorContains(t, err, "data key")
}

func TestDotenvMetadataAndINI(t *testing.T) {
	_, err := Decode([]byte("password=hunter2\nsops_mac=ENC[nope]\n"))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotAge)

	_, err = Decode([]byte("[sops]\nmac=ENC[nope]\n"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "ini")
}

func TestPlainDotenvPassesThrough(t *testing.T) {
	raw := []byte("# keep\npassword=hunter2\n")
	got, err := Decode(raw)
	require.NoError(t, err)
	require.Equal(t, raw, got)

	dir := t.TempDir()
	path := filepath.Join(dir, "file.env")
	require.NoError(t, os.WriteFile(path, raw, 0o600))
	got, err = Open(path)
	require.NoError(t, err)
	require.Equal(t, raw, got)
}

func TestAgeKeyCommand(t *testing.T) {
	isolateAge(t)
	path, err := filepath.Abs("testdata/age.txt")
	require.NoError(t, err)
	t.Setenv("SOPS_AGE_KEY_CMD", "cat "+path)
	want, err := os.ReadFile("testdata/blob.plain")
	require.NoError(t, err)
	got, err := Open("testdata/blob.sops")
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func useAgeKey(t *testing.T) {
	t.Helper()
	isolateAge(t)
	key, err := os.ReadFile("testdata/age.txt")
	require.NoError(t, err)
	t.Setenv("SOPS_AGE_KEY", string(key))
}

func isolateAge(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, key := range []string{
		"SOPS_AGE_KEY",
		"SOPS_AGE_KEY_FILE",
		"SOPS_AGE_KEY_CMD",
		"SOPS_AGE_SSH_PRIVATE_KEY_FILE",
	} {
		unsetEnv(t, key)
	}
}

func unsetEnv(t *testing.T, key string) {
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
