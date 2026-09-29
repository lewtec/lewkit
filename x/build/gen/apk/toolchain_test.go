package apk

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/stretchr/testify/require"
)

func TestJDKVersionOK(t *testing.T) {
	require.False(t, jdkVersionOK("openjdk version \"17.0.2\" 2022-01-18\nOpenJDK Runtime Environment (build 17.0.2+8)"))
	require.True(t, jdkVersionOK("openjdk version \"17.0.14\" 2025-01-21\nOpenJDK Runtime Environment Temurin-17.0.14+7"))
	require.True(t, jdkVersionOK("openjdk version \"21.0.6\" 2025-01-21"))
	require.False(t, jdkVersionOK("openjdk version \"11.0.16\""))
	require.False(t, jdkVersionOK("not a jdk"))
}

func TestPickNDKPrefersKnown(t *testing.T) {
	sdk := t.TempDir()
	host := ndkHost()
	clang := "aarch64-linux-android21-clang"
	writeClang(t, sdk, "26.1.10909125")
	writeClang(t, sdk, preferredNDK)
	got, err := pickNDK(sdk, host, clang)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(sdk, "ndk", preferredNDK), got)
}

func TestPickNDKUsesOtherWhenPreferredMissing(t *testing.T) {
	sdk := t.TempDir()
	host := ndkHost()
	clang := "aarch64-linux-android21-clang"
	writeClang(t, sdk, "26.1.10909125")
	got, err := pickNDK(sdk, host, clang)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(sdk, "ndk", "26.1.10909125"), got)
}

func TestPickNDKMissingNamesPackage(t *testing.T) {
	sdk := t.TempDir()
	_, err := pickNDK(sdk, ndkHost(), "aarch64-linux-android21-clang")
	require.Error(t, err)
	require.ErrorContains(t, err, preferredNDK)
}

func TestFindGradleNestedLauncher(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "gradle-9.6.1", "bin", "gradle")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755))
	require.Equal(t, path, findGradle(root))
}

func TestFindGradleBinDir(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "bin", "gradle")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755))
	require.Equal(t, path, findGradle(root))
}

func TestGradleEnvOverridesInheritedJDK(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")
	env := gradleEnv([]string{"JAVA_HOME=/bad/17.0.2", "PATH=/usr/bin"}, "/jdk/temurin", "/sdk")
	require.Equal(t, "/jdk/temurin", lastEnv(env, "JAVA_HOME"))
	require.Equal(t, "/sdk", lastEnv(env, "ANDROID_HOME"))
	require.True(t, strings.HasPrefix(lastEnv(env, "PATH"), "/jdk/temurin/bin"+string(os.PathListSeparator)))
}

func TestNDKNewerMajor(t *testing.T) {
	require.True(t, ndkNewer("27.2.12479018", "9.0.0"))
	require.False(t, ndkNewer("9.0.0", "27.2.12479018"))
}

func TestAndroidSDKEnvWins(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ANDROID_HOME", dir)
	t.Setenv("ANDROID_SDK_ROOT", "")
	got, err := androidSDK(t.Context())
	require.NoError(t, err)
	require.Equal(t, dir, got)
}

func TestAndroidSDKRejectsNonDir(t *testing.T) {
	t.Setenv("ANDROID_HOME", filepath.Join(t.TempDir(), "missing"))
	_, err := androidSDK(t.Context())
	require.ErrorIs(t, err, ErrSDKEnvNotDir)
}

func TestMiseToolchain(t *testing.T) {
	if _, err := execdriver.Which(t.Context(), "mise"); err != nil {
		t.Skip("mise")
	}
	t.Setenv("JAVA_HOME", t.TempDir())
	t.Setenv("ANDROID_HOME", "")
	t.Setenv("ANDROID_SDK_ROOT", "")
	t.Setenv("ANDROID_NDK_HOME", "")
	t.Setenv("ANDROID_NDK_ROOT", "")
	t.Setenv("NDK_HOME", "")
	t.Setenv("CC", "")

	ctx := t.Context()
	home, err := javaHome(ctx)
	require.NoError(t, err)
	require.NotEqual(t, os.Getenv("JAVA_HOME"), home)
	ok, err := jdkDirOK(ctx, home)
	require.NoError(t, err)
	require.True(t, ok)

	sdk, err := androidSDK(ctx)
	require.NoError(t, err)
	require.True(t, isDir(sdk))

	// android-sdk@13.0 is command-line tools. adb and the NDK are separate.
	cc, err := ndkCC(ctx, "arm64")
	if err != nil {
		require.ErrorIs(t, err, errNDKMissing)
		require.ErrorContains(t, err, preferredNDK)
	} else {
		require.Contains(t, cc, "aarch64-linux-android21-clang")
	}

	root, err := miseWhere(ctx, miseGradleSpec)
	if err != nil {
		return
	}
	require.NotEmpty(t, findGradle(root))
}

func lastEnv(env []string, key string) string {
	prefix := key + "="
	var got string
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			got = strings.TrimPrefix(item, prefix)
		}
	}
	return got
}

func writeClang(t *testing.T, sdk, version string) {
	t.Helper()
	path := filepath.Join(sdk, "ndk", version, "toolchains", "llvm", "prebuilt", ndkHost(), "bin", "aarch64-linux-android21-clang")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, nil, 0o755))
}
