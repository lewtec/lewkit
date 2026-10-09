package github

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	keyringGet = func(string, string) (string, error) {
		return "", keyring.ErrNotFound
	}
	os.Exit(m.Run())
}

func TestResolveTokenEnvOrder(t *testing.T) {
	t.Setenv("GH_TOKEN", "from-gh-token")
	t.Setenv("GITHUB_TOKEN", "from-github-token")
	t.Setenv("GH_CONFIG_DIR", t.TempDir())

	got := resolveToken(t.Context())
	require.Equal(t, "from-gh-token", got)
}

func TestResolveTokenSkipsStop(t *testing.T) {
	t.Setenv("GH_TOKEN", "STOP")
	t.Setenv("GITHUB_TOKEN", "from-github-token")
	t.Setenv("GH_CONFIG_DIR", t.TempDir())

	got := resolveToken(t.Context())
	require.Equal(t, "from-github-token", got)
}

func TestResolveTokenHostsBeforeKeyring(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	writeHosts(t, dir, `
github.com:
  user: monalisa
  oauth_token: from-file
  users:
    monalisa:
      oauth_token: from-user
`)
	lookedUp := false
	keyringGet = func(string, string) (string, error) {
		lookedUp = true
		return "", keyring.ErrNotFound
	}
	t.Cleanup(restoreKeyring)

	got := resolveToken(t.Context())
	require.Equal(t, "from-file", got)
	require.False(t, lookedUp, "keyring looked up after hosts.yml had a token")
}

func TestResolveTokenUserTokenWhenHostTokenMissing(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	writeHosts(t, dir, `
github.com:
  user: monalisa
  users:
    monalisa:
      oauth_token: from-user
    other:
      oauth_token: from-other
`)

	got := resolveToken(t.Context())
	require.Equal(t, "from-user", got)
}

func TestResolveTokenKeyringBeforeGH(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	var service, user string
	keyringGet = func(gotService, gotUser string) (string, error) {
		service, user = gotService, gotUser
		return "from-keyring", nil
	}
	t.Cleanup(restoreKeyring)

	got := resolveToken(t.Context())
	require.Equal(t, "from-keyring", got)
	require.Equal(t, ghKeyringService, service)
	require.Empty(t, user)
}

func TestResolveTokenKeyringUsesActiveUser(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	writeHosts(t, dir, `
github.com:
  user: monalisa
`)
	var accounts []string
	var service string
	keyringGet = func(gotService, user string) (string, error) {
		service = gotService
		accounts = append(accounts, user)
		if user == "monalisa" {
			return "from-user-keyring", nil
		}
		return "", keyring.ErrNotFound
	}
	t.Cleanup(restoreKeyring)

	got := resolveToken(t.Context())
	require.Equal(t, "from-user-keyring", got)
	require.Equal(t, ghKeyringService, service)
	require.Equal(t, []string{"monalisa"}, accounts)
}

func TestResolveTokenContinuesAfterBadHosts(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	writeHosts(t, dir, "github.com: [\n")
	keyringGet = func(string, string) (string, error) {
		return "from-keyring", nil
	}
	t.Cleanup(restoreKeyring)

	got := resolveToken(t.Context())
	require.Equal(t, "from-keyring", got)
}

func TestTokenFromHostsXDGConfigHome(t *testing.T) {
	t.Setenv("GH_CONFIG_DIR", "")
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	writeHosts(t, filepath.Join(xdg, "gh"), `
github.com:
  oauth_token: from-xdg
`)

	got, err := tokenFromHosts(t.Context())
	require.NoError(t, err)
	require.Equal(t, "from-xdg", got)
}

func TestTokenFromHostsMissingFile(t *testing.T) {
	t.Setenv("GH_CONFIG_DIR", t.TempDir())

	got, err := tokenFromHosts(t.Context())
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestTokenFromHostsRejectsBadYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	writeHosts(t, dir, "github.com: [\n")

	_, err := tokenFromHosts(t.Context())
	require.Error(t, err)
}

func TestLookupSkipsProbe(t *testing.T) {
	t.Setenv(tokenProbeEnv, tokenProbeVal)
	t.Setenv("GITHUB_TOKEN", "from-env")

	require.Empty(t, Lookup(t.Context()))
	require.Empty(t, Token(t.Context()))
}

func restoreKeyring() {
	keyringGet = func(string, string) (string, error) {
		return "", keyring.ErrNotFound
	}
}

func writeHosts(t *testing.T, dir, body string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hosts.yml"), []byte(body), 0o600))
}
