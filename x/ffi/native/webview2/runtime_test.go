package webview2

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientFileUsesTheInstallFolder(t *testing.T) {
	install := `C:\Program Files (x86)\Microsoft\EdgeWebView\Application\120.0.2210.91`
	got, ok := clientFile(install, "x64")
	require.True(t, ok)
	require.Equal(t, install+`\EBWebView\x64\EmbeddedBrowserWebView.dll`, got)
}

func TestClientFileAcceptsTheMinimumRuntime(t *testing.T) {
	got, ok := clientFile(`C:\WV\Application\86.0.616.0`, "arm64")
	require.True(t, ok)
	require.Equal(t, `C:\WV\Application\86.0.616.0\EBWebView\arm64\EmbeddedBrowserWebView.dll`, got)
}

func TestClientFileRejectsAnOlderRuntime(t *testing.T) {
	_, ok := clientFile(`C:\WV\Application\86.0.615.9`, "x64")
	require.False(t, ok)
	_, ok = clientFile(`C:\WV\Application\85.9.9999.9`, "x86")
	require.False(t, ok)
}

func TestClientFileRejectsAShortVersion(t *testing.T) {
	_, ok := clientFile(`C:\WV\Application\120.0.2210`, "x64")
	require.False(t, ok)
}

func TestArchName(t *testing.T) {
	require.Equal(t, "x64", archName("amd64"))
	require.Equal(t, "x86", archName("386"))
	require.Equal(t, "arm64", archName("arm64"))
	require.Equal(t, "", archName("riscv64"))
}

func TestFindClientPrefersCurrentUserStable(t *testing.T) {
	stable := `C:\Program Files (x86)\Microsoft\EdgeWebView\Application\120.0.2210.91`
	canary := `C:\Users\me\AppData\Local\Microsoft\Edge\Application\121.0.2277.4`
	got, err := findClient(func(hive, guid string) string {
		switch {
		case hive == hiveCurrentUser && guid == evergreenChannels[0]:
			return stable
		case hive == hiveCurrentUser && guid == evergreenChannels[3]:
			return canary
		default:
			return ""
		}
	}, "x64", func(string) bool { return true })
	require.NoError(t, err)
	require.Equal(t, stable+`\EBWebView\x64\EmbeddedBrowserWebView.dll`, got)
}

func TestFindClientFallsThroughToLocalMachine(t *testing.T) {
	folder := `C:\Program Files (x86)\Microsoft\EdgeWebView\Application\131.0.2903.86`
	got, err := findClient(func(hive, guid string) string {
		if hive == hiveLocalMachine && guid == evergreenChannels[0] {
			return folder
		}
		return ""
	}, "x64", func(string) bool { return true })
	require.NoError(t, err)
	require.Equal(t, folder+`\EBWebView\x64\EmbeddedBrowserWebView.dll`, got)
}

func TestFindClientSkipsAnOldRuntime(t *testing.T) {
	newer := `C:\Program Files (x86)\Microsoft\EdgeWebView\Application\99.0.1150.46`
	got, err := findClient(func(hive, guid string) string {
		if hive == hiveCurrentUser && guid == evergreenChannels[0] {
			return `C:\Old\86.0.615.0`
		}
		if hive == hiveLocalMachine && guid == evergreenChannels[1] {
			return newer
		}
		return ""
	}, "x64", func(string) bool { return true })
	require.NoError(t, err)
	require.Equal(t, newer+`\EBWebView\x64\EmbeddedBrowserWebView.dll`, got)
}

func TestFindClientSkipsAMissingDLL(t *testing.T) {
	first := `C:\Program Files (x86)\Microsoft\EdgeWebView\Application\120.0.2210.91`
	second := `C:\Program Files (x86)\Microsoft\EdgeWebView\Application\121.0.2277.4`
	got, err := findClient(func(hive, guid string) string {
		if hive == hiveCurrentUser && guid == evergreenChannels[0] {
			return first
		}
		if hive == hiveLocalMachine && guid == evergreenChannels[0] {
			return second
		}
		return ""
	}, "x64", func(path string) bool {
		return path == second+`\EBWebView\x64\EmbeddedBrowserWebView.dll`
	})
	require.NoError(t, err)
	require.Equal(t, second+`\EBWebView\x64\EmbeddedBrowserWebView.dll`, got)
}

func TestFindClientReportsAMissingRuntime(t *testing.T) {
	_, err := findClient(func(string, string) string { return "" }, "x64", func(string) bool { return false })
	require.ErrorIs(t, err, ErrUnavailable)
	require.ErrorContains(t, err, "not installed")

	_, err = findClient(func(string, string) string { return "" }, "", func(string) bool { return true })
	require.ErrorIs(t, err, ErrUnavailable)

	_, err = findClient(func(hive, guid string) string {
		if hive == hiveCurrentUser && guid == evergreenChannels[0] {
			return `C:\Old\86.0.100.0`
		}
		return ""
	}, "x64", func(string) bool { return true })
	require.ErrorIs(t, err, ErrUnavailable)
	require.ErrorContains(t, err, "older than 86.0.616.0")
}

func TestEvergreenChannelsStartAtStable(t *testing.T) {
	require.Equal(t, "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}", evergreenChannels[0])
	require.Equal(t, "{BE59E8FD-089A-411B-A3B0-051D9E417818}", evergreenChannels[len(evergreenChannels)-1])
}
