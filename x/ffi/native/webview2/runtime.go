package webview2

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const minWebViewVersion = "86.0.616.0"

const embeddedBrowserDLL = "EmbeddedBrowserWebView.dll"

// Stable, then beta, dev, and canary. EdgeUpdate records these clients.
var channelGUID = []string{
	"{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}",
	"{2CD8A007-E189-409D-A2C8-9AF4EF3C72AA}",
	"{0D50BFEC-CD6A-4F9A-964C-C7416E3ACB10}",
	"{65C35B14-6C1D-4122-AC46-7148CC9D6497}",
}

const (
	clientStatePrefix = `Software\Microsoft\EdgeUpdate\ClientState\`
	clientsPrefix     = `Software\Microsoft\EdgeUpdate\Clients\`
)

var (
	errOldRuntime  = errors.New("webview runtime is too old")
	errNoRuntime   = errors.New("edge webview runtime is not installed")
	errRuntimeFile = errors.New("webview runtime dll missing")
)

func webViewArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	case "386":
		return "x86"
	default:
		return ""
	}
}

// runtimeDLL is the browser control inside an Edge WebView install folder.
func runtimeDLL(folder, arch string) string {
	return filepath.Join(folder, "EBWebView", arch, embeddedBrowserDLL)
}

// acceptInstall returns the runtime DLL when folder's version is new enough
// and that file exists. folder is the version directory, not EBWebView itself.
func acceptInstall(folder, arch string) (string, error) {
	ver := filepath.Base(folder)
	if !versionAtLeast(ver, minWebViewVersion) {
		return "", fmt.Errorf("%w: %s", errOldRuntime, ver)
	}
	if arch == "" {
		return "", fmt.Errorf("%w: empty arch", errRuntimeFile)
	}
	path := runtimeDLL(folder, arch)
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("%w: %s", errRuntimeFile, path)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%w: %s", errRuntimeFile, path)
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("%w: %s", errRuntimeFile, path)
	}
	return path, nil
}

func versionAtLeast(got, min string) bool {
	g := parseQuad(got)
	m := parseQuad(min)
	for i := range g {
		if g[i] == m[i] {
			continue
		}
		return g[i] > m[i]
	}
	return true
}

func parseQuad(s string) [4]int {
	var out [4]int
	parts := strings.Split(s, ".")
	for i := 0; i < len(parts) && i < len(out); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return [4]int{}
		}
		out[i] = n
	}
	return out
}
