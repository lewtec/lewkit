package apk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/tool"
)

// mise specs for the Android host.
// java@17 resolves to OpenJDK 17.0.2, which throws in CgroupV2Subsystem.
const (
	miseJavaSpec   = "java@temurin-17.0.14+7"
	miseSDKSpec    = "android-sdk@13.0"
	miseGradleSpec = "gradle@9.6.1"
	miseYQSpec     = "yq@4.50.1"
	preferredNDK   = "27.2.12479018"
)

var (
	errJDK       = errors.New("need JDK 17+ other than OpenJDK 17.0.2 (mise " + miseJavaSpec + ")")
	errMiseEmpty = errors.New("mise where returned no path")
	jdkVersionR  = regexp.MustCompile(`version "(\d+)(?:\.(\d+))?(?:\.(\d+))?`)
)

// AndroidSDK returns the Android SDK root for Gradle and adb.
// ANDROID_HOME and ANDROID_SDK_ROOT win when they name a directory.
// Otherwise mise android-sdk@13.0, then the usual Android Studio paths.
func AndroidSDK(ctx context.Context) (string, error) {
	return androidSDK(ctx)
}

func androidSDK(ctx context.Context) (string, error) {
	if dir, err := sdkFromEnv(); err != nil || dir != "" {
		return dir, err
	}
	if root, err := miseWhere(ctx, miseSDKSpec); err == nil && isDir(root) {
		slog.Info("android sdk", "dir", root)
		return root, nil
	}
	if dir, ok := sdkFromHome(); ok {
		return dir, nil
	}
	root, err := miseInstallWhere(ctx, miseSDKSpec)
	if err != nil || !isDir(root) {
		if err == nil {
			err = fmt.Errorf("%w: %s", errMiseEmpty, miseSDKSpec)
		}
		return "", fmt.Errorf("%w: %w", ErrAndroidSDKNotFound, err)
	}
	slog.Info("android sdk", "dir", root)
	return root, nil
}

func sdkFromEnv() (string, error) {
	for _, key := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		if !isDir(value) {
			return "", fmt.Errorf("%w: %s=%q", ErrSDKEnvNotDir, key, value)
		}
		return value, nil
	}
	return "", nil
}

func sdkFromHome() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false
	}
	for _, rel := range []string{
		"Android/Sdk",
		"Library/Android/sdk",
		"AppData/Local/Android/Sdk",
	} {
		dir := filepath.Join(home, rel)
		if isDir(dir) {
			return dir, true
		}
	}
	return "", false
}

func javaHome(ctx context.Context) (string, error) {
	if home := strings.TrimSpace(os.Getenv("JAVA_HOME")); home != "" {
		ok, reason := jdkDirOK(ctx, home)
		if ok {
			return home, nil
		}
		slog.Warn("android jdk", "JAVA_HOME", home, "err", reason)
	}
	if bin, err := execdriver.Which(ctx, "java"); err == nil {
		home := javaInstallRoot(bin)
		ok, reason := jdkDirOK(ctx, home)
		if ok {
			return home, nil
		}
		slog.Warn("android jdk", "home", home, "err", reason)
	}
	root, err := miseWhere(ctx, miseJavaSpec)
	if err != nil || !isDir(root) {
		root, err = miseInstallWhere(ctx, miseJavaSpec)
	}
	if err != nil {
		return "", fmt.Errorf("%w: %w", errJDK, err)
	}
	ok, err := jdkDirOK(ctx, root)
	if !ok {
		return "", fmt.Errorf("%w: %s: %w", errJDK, root, err)
	}
	slog.Info("android jdk", "home", root)
	return root, nil
}

func javaInstallRoot(bin string) string {
	resolved, err := filepath.EvalSymlinks(bin)
	if err != nil {
		resolved = bin
	}
	return filepath.Dir(filepath.Dir(resolved))
}

func jdkDirOK(ctx context.Context, home string) (bool, error) {
	bin := filepath.Join(home, "bin", "java")
	if !isFile(bin) {
		return false, fmt.Errorf("%w: %s", os.ErrNotExist, bin)
	}
	text, err := javaVersionText(ctx, bin)
	if err != nil {
		return false, err
	}
	if !jdkVersionOK(text) {
		return false, fmt.Errorf("%w: %s", errJDK, strings.TrimSpace(text))
	}
	return true, nil
}

func javaVersionText(ctx context.Context, bin string) (string, error) {
	command := execdriver.MustCommand(bin, "-version")
	var buf bytes.Buffer
	command.Stdout = &buf
	command.Stderr = &buf
	if err := execdriver.Run(ctx, command); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// jdkVersionOK reports whether java -version output is JDK 17 or newer
// and is not OpenJDK 17.0.2.
func jdkVersionOK(text string) bool {
	major, minor, patch, ok := jdkVersion(text)
	if !ok || major < 17 {
		return false
	}
	return !(major == 17 && minor == 0 && patch == 2)
}

func jdkVersion(text string) (major, minor, patch int, ok bool) {
	match := jdkVersionR.FindStringSubmatch(text)
	if match == nil {
		return 0, 0, 0, false
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, 0, false
	}
	if match[2] != "" {
		minor, err = strconv.Atoi(match[2])
		if err != nil {
			return 0, 0, 0, false
		}
	}
	if match[3] != "" {
		patch, err = strconv.Atoi(match[3])
		if err != nil {
			return 0, 0, 0, false
		}
	}
	return major, minor, patch, true
}

func resolveGradle(ctx context.Context, workDir string) ([]string, error) {
	wrapper := filepath.Join(workDir, "gradlew")
	if isFile(wrapper) {
		return []string{wrapper}, nil
	}
	if bin, err := execdriver.Which(ctx, "gradle"); err == nil {
		return []string{bin}, nil
	}
	root, err := miseWhere(ctx, miseGradleSpec)
	if err != nil || !isDir(root) {
		root, err = miseInstallWhere(ctx, miseGradleSpec)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGradleNotFound, err)
	}
	bin := findGradle(root)
	if bin == "" {
		return nil, fmt.Errorf("%w: no launcher under %s", ErrGradleNotFound, root)
	}
	slog.Info("android gradle", "bin", bin)
	return []string{bin}, nil
}

// findGradle locates the gradle launcher. mise gradle@9.6.1 keeps it at
// gradle-<version>/bin/gradle, not at the install root.
func findGradle(root string) string {
	if bin := tool.FindBinary(root, "gradle"); bin != "" {
		return bin
	}
	for _, name := range []string{"gradle", "gradle.bat"} {
		matches, err := filepath.Glob(filepath.Join(root, "*", "bin", name))
		if err != nil || len(matches) == 0 {
			continue
		}
		sort.Strings(matches)
		return matches[len(matches)-1]
	}
	return ""
}

func gradleEnv(base []string, javaHome, sdk string) []string {
	path := os.Getenv("PATH")
	return append(base,
		"JAVA_HOME="+javaHome,
		"ANDROID_HOME="+sdk,
		"ANDROID_SDK_ROOT="+sdk,
		"PATH="+filepath.Join(javaHome, "bin")+string(os.PathListSeparator)+path,
	)
}

func miseWhere(ctx context.Context, spec string) (string, error) {
	bin, err := execdriver.Which(ctx, "mise")
	if err != nil {
		return "", err
	}
	return miseWhereBin(ctx, bin, spec)
}

func miseWhereBin(ctx context.Context, miseBin, spec string) (string, error) {
	command := execdriver.MustCommand(miseBin, "where", spec)
	out, err := execdriver.Output(ctx, command)
	if err != nil {
		return "", err
	}
	var found string
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "ignoring leftover") {
			continue
		}
		if filepath.IsAbs(line) {
			found = line
		}
	}
	if found == "" {
		return "", fmt.Errorf("%w: %s", errMiseEmpty, spec)
	}
	return found, nil
}

func miseInstallWhere(ctx context.Context, spec string) (string, error) {
	bin, err := execdriver.Which(ctx, "mise")
	if err != nil {
		return "", err
	}
	command := execdriver.MustCommand(bin, "install", spec)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if yq := yqDir(ctx, bin); yq != "" {
		command.Env = append(os.Environ(), "PATH="+yq+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	if err := execdriver.Run(ctx, command); err != nil {
		return "", err
	}
	return miseWhereBin(ctx, bin, spec)
}

func yqDir(ctx context.Context, miseBin string) string {
	if bin, err := execdriver.Which(ctx, "yq"); err == nil {
		return filepath.Dir(bin)
	}
	root, err := miseWhereBin(ctx, miseBin, miseYQSpec)
	if err != nil {
		return ""
	}
	if bin := tool.FindBinary(root, "yq"); bin != "" {
		return filepath.Dir(bin)
	}
	return ""
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
