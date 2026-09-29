package apk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var errNDKMissing = errors.New("android cgo needs an NDK")

// ndkCC is the NDK clang for goarch. ANDROID_NDK_HOME, ANDROID_NDK_ROOT,
// and NDK_HOME name the NDK. Otherwise the clang is taken from an NDK
// installed under the Android SDK (prefer ndk;27.2.12479018).
// ANDROID_API selects the platform (default 21).
// arm64 uses aarch64-linux-android, which is the arm64-v8a ABI.
func ndkCC(ctx context.Context, goarch string) (string, error) {
	api := strings.TrimSpace(os.Getenv("ANDROID_API"))
	if api == "" {
		api = "21"
	}
	triple, err := ndkTriple(goarch)
	if err != nil {
		return "", err
	}
	clang := triple + api + "-clang"
	if root, ok := ndkFromEnv(); ok {
		return clangAt(root, clang)
	}
	if sdk, err := androidSDK(ctx); err == nil {
		root, pickErr := pickNDK(sdk, ndkHost(), clang)
		if pickErr == nil {
			slog.Info("android ndk", "dir", root)
			return clangAt(root, clang)
		}
		if strings.TrimSpace(os.Getenv("CC")) == "" {
			return "", pickErr
		}
	}
	if cc := strings.TrimSpace(os.Getenv("CC")); cc != "" {
		return cc, nil
	}
	return "", fmt.Errorf("%w: ANDROID_NDK_HOME or ndk;%s (%s)", errNDKMissing, preferredNDK, clang)
}

func ndkFromEnv() (string, bool) {
	for _, key := range []string{"ANDROID_NDK_HOME", "ANDROID_NDK_ROOT", "NDK_HOME"} {
		if dir := strings.TrimSpace(os.Getenv(key)); dir != "" {
			return dir, true
		}
	}
	return "", false
}

func clangAt(root, clang string) (string, error) {
	bin := filepath.Join(root, "toolchains", "llvm", "prebuilt", ndkHost(), "bin", clang)
	if _, err := os.Stat(bin); err != nil {
		return "", fmt.Errorf("ndk clang %s: %w", bin, err)
	}
	return bin, nil
}

// pickNDK chooses an NDK directory under sdk/ndk that contains clang.
// preferredNDK wins when it is present. Otherwise the highest version wins.
func pickNDK(sdk, host, clang string) (string, error) {
	base := filepath.Join(sdk, "ndk")
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", fmt.Errorf("%w: ndk;%s (%s): %w", errNDKMissing, preferredNDK, clang, err)
	}
	var hits []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bin := filepath.Join(base, entry.Name(), "toolchains", "llvm", "prebuilt", host, "bin", clang)
		info, err := os.Stat(bin)
		if err != nil || info.IsDir() {
			continue
		}
		if entry.Name() == preferredNDK {
			return filepath.Join(base, entry.Name()), nil
		}
		hits = append(hits, entry.Name())
	}
	if len(hits) == 0 {
		return "", fmt.Errorf("%w: ndk;%s (%s)", errNDKMissing, preferredNDK, clang)
	}
	best := hits[0]
	for _, name := range hits[1:] {
		if ndkNewer(name, best) {
			best = name
		}
	}
	return filepath.Join(base, best), nil
}

func ndkNewer(a, b string) bool {
	ai, aerr := leadingInt(a)
	bi, berr := leadingInt(b)
	if aerr == nil && berr == nil && ai != bi {
		return ai > bi
	}
	return a > b
}

func leadingInt(value string) (int, error) {
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, fmt.Errorf("%w: %q", strconv.ErrSyntax, value)
	}
	return strconv.Atoi(value[:end])
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
