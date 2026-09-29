package webview2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcceptInstall(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "120.0.2210.91")
	dll := runtimeDLL(current, "x64")
	require.NoError(t, os.MkdirAll(filepath.Dir(dll), 0o755))
	require.NoError(t, os.WriteFile(dll, []byte("dll"), 0o644))

	got, err := acceptInstall(current, "x64")
	require.NoError(t, err)
	require.Equal(t, dll, got)

	old := filepath.Join(root, "85.0.0.0")
	oldDLL := runtimeDLL(old, "x64")
	require.NoError(t, os.MkdirAll(filepath.Dir(oldDLL), 0o755))
	require.NoError(t, os.WriteFile(oldDLL, []byte("dll"), 0o644))
	_, err = acceptInstall(old, "x64")
	require.ErrorIs(t, err, errOldRuntime)

	_, err = acceptInstall(filepath.Join(root, "130.0.1.2"), "x64")
	require.ErrorIs(t, err, errRuntimeFile)
}

func TestVersionAtLeast(t *testing.T) {
	require.True(t, versionAtLeast("86.0.616.0", minWebViewVersion))
	require.True(t, versionAtLeast("120.0.2210.91", minWebViewVersion))
	require.False(t, versionAtLeast("86.0.615.9", minWebViewVersion))
	require.False(t, versionAtLeast("not-a-version", minWebViewVersion))
}

func TestChannelOrder(t *testing.T) {
	require.Equal(t, "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}", channelGUID[0])
	require.Equal(t, `Software\Microsoft\EdgeUpdate\ClientState\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`, clientStatePrefix+channelGUID[0])
	require.Equal(t, `Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`, clientsPrefix+channelGUID[0])
	require.Equal(t, "x64", webViewArch("amd64"))
	require.Equal(t, "arm64", webViewArch("arm64"))
	require.Equal(t, "x86", webViewArch("386"))
	require.Empty(t, webViewArch("ppc64"))
}

func TestRuntimeDLL(t *testing.T) {
	got := runtimeDLL(`C:\Program Files (x86)\Microsoft\EdgeWebView\Application\120.0.2210.91`, "x64")
	require.True(t, strings.Contains(got, "EBWebView"))
	require.True(t, strings.HasSuffix(got, embeddedBrowserDLL))
}
