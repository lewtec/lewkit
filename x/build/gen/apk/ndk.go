package apk

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ndkCC is the NDK clang for goarch. ANDROID_NDK_HOME, ANDROID_NDK_ROOT,
// and NDK_HOME name the NDK. ANDROID_API selects the platform (default 21).
// arm64 uses aarch64-linux-android, which is the arm64-v8a ABI.
func ndkCC(goarch string) (string, error) {
	api := strings.TrimSpace(os.Getenv("ANDROID_API"))
	if api == "" {
		api = "21"
	}
	triple, err := ndkTriple(goarch)
	if err != nil {
		return "", err
	}
	name := triple + api + "-clang"
	root := ndkRoot()
	if root == "" {
		if cc := strings.TrimSpace(os.Getenv("CC")); cc != "" {
			return cc, nil
		}
		return "", fmt.Errorf("android cgo needs ANDROID_NDK_HOME (or CC) for %s", name)
	}
	bin := filepath.Join(root, "toolchains", "llvm", "prebuilt", ndkHost(), "bin", name)
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("ndk clang %s: %w", bin, err)
	}
	return bin, nil
}

func ndkRoot() string {
	for _, key := range []string{"ANDROID_NDK_HOME", "ANDROID_NDK_ROOT", "NDK_HOME"} {
		if dir := strings.TrimSpace(os.Getenv(key)); dir != "" {
			return dir
		}
	}
	return ""
}

func ndkHost() string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return "darwin-arm64"
		}
		return "darwin-x86_64"
	case "windows":
		return "windows-x86_64"
	default:
		if runtime.GOARCH == "arm64" {
			return "linux-aarch64"
		}
		return "linux-x86_64"
	}
}

func ndkTriple(goarch string) (string, error) {
	switch goarch {
	case "arm64":
		return "aarch64-linux-android", nil
	case "arm":
		return "armv7a-linux-androideabi", nil
	case "amd64":
		return "x86_64-linux-android", nil
	case "386":
		return "i686-linux-android", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedABI, goarch)
	}
}
