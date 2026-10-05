package webview2

import (
	"fmt"
	"strconv"
	"strings"
)

// evergreenChannels is the Edge updater client id, stable first.
// The install folder is the EBWebView value of
// Software\Microsoft\EdgeUpdate\ClientState\<guid>, in the 32-bit registry view.
var evergreenChannels = []string{
	"{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}", // stable
	"{2CD8A007-E189-409D-A2C8-9AF4EF3C72AA}", // beta
	"{0D50BFEC-CD6A-4F9A-964C-C7416E3ACB10}", // dev
	"{65C35B14-6C1D-4122-AC46-7148CC9D6497}", // canary
	"{BE59E8FD-089A-411B-A3B0-051D9E417818}", // internal
}

const (
	hiveCurrentUser  = "HKCU"
	hiveLocalMachine = "HKLM"

	clientLibrary = "EmbeddedBrowserWebView.dll"
)

// minimumRuntime is the oldest Edge WebView2 the loader accepts.
var minimumRuntime = runtimeVersion{86, 0, 616, 0}

type runtimeVersion [4]uint32

func archName(goarch string) string {
	switch goarch {
	case "amd64":
		return "x64"
	case "386":
		return "x86"
	case "arm64":
		return "arm64"
	default:
		return ""
	}
}

// clientFile is EBWebView\<arch>\EmbeddedBrowserWebView.dll inside an install
// folder whose last path component is a version of at least 86.0.616.0.
func clientFile(install, arch string) (string, bool) {
	if arch == "" {
		return "", false
	}
	version, ok := folderVersion(install)
	if !ok || !version.atLeast(minimumRuntime) {
		return "", false
	}
	base := strings.TrimRight(install, `\`)
	return base + `\EBWebView\` + arch + `\` + clientLibrary, true
}

func folderVersion(install string) (runtimeVersion, bool) {
	install = strings.TrimRight(install, `\`)
	slash := strings.LastIndex(install, `\`)
	if slash < 0 || slash+1 >= len(install) {
		return runtimeVersion{}, false
	}
	return parseVersion(install[slash+1:])
}

func parseVersion(text string) (runtimeVersion, bool) {
	var version runtimeVersion
	rest := text
	for i := range version {
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if end == 0 {
			return runtimeVersion{}, false
		}
		n, err := strconv.ParseUint(rest[:end], 10, 32)
		if err != nil {
			return runtimeVersion{}, false
		}
		version[i] = uint32(n)
		rest = rest[end:]
		if i == len(version)-1 {
			return version, true
		}
		if rest == "" || rest[0] != '.' {
			return runtimeVersion{}, false
		}
		rest = rest[1:]
	}
	return version, true
}

func (version runtimeVersion) atLeast(min runtimeVersion) bool {
	for i := range version {
		if version[i] < min[i] {
			return false
		}
		if version[i] > min[i] {
			return true
		}
	}
	return true
}

// findClient walks stable, beta, dev, canary, then internal. Each channel is
// read from HKCU and then HKLM. lookup returns the EBWebView folder, or empty
// when that key has no install. exists reports that the client DLL is present.
func findClient(lookup func(hive, guid string) string, arch string, exists func(string) bool) (string, error) {
	if arch == "" {
		return "", fmt.Errorf("%w: unsupported architecture", ErrUnavailable)
	}
	var missing []string
	sawOld := false
	for _, guid := range evergreenChannels {
		for _, hive := range []string{hiveCurrentUser, hiveLocalMachine} {
			folder := strings.TrimSpace(lookup(hive, guid))
			if folder == "" {
				continue
			}
			if version, ok := folderVersion(folder); ok && !version.atLeast(minimumRuntime) {
				sawOld = true
				continue
			}
			path, ok := clientFile(folder, arch)
			if !ok {
				continue
			}
			if exists(path) {
				return path, nil
			}
			missing = append(missing, path)
		}
	}
	if len(missing) > 0 {
		return "", fmt.Errorf("%w: %s", ErrUnavailable, strings.Join(missing, "; "))
	}
	if sawOld {
		return "", fmt.Errorf("%w: Edge WebView2 runtime is older than 86.0.616.0", ErrUnavailable)
	}
	return "", fmt.Errorf("%w: Edge WebView2 runtime is not installed", ErrUnavailable)
}
