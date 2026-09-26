package apk

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/build/version"
)

// Sentinel errors for static build/setup failures (errors.Is / wrap with %w).
var (
	ErrOutAPKRequired     = errors.New("out apk path is required (or use --go-only)")
	ErrUnsupportedABI     = errors.New("unsupported abi")
	ErrDebugAPKMissing    = errors.New("gradle succeeded but no debug APK under app/build/outputs/apk/debug")
	ErrSDKEnvNotDir       = errors.New("android SDK env path is not a directory")
	ErrAndroidSDKNotFound = errors.New("android SDK not found: set ANDROID_HOME (or ANDROID_SDK_ROOT) to the SDK root")
	ErrGradleNotFound     = errors.New("neither ./gradlew nor gradle on PATH; install Gradle 8.9+ (or Android Studio) and JDK 17")
)

func applyIconMipmaps(iconRoot, androidResDir string) error {
	return icons.ApplyAndroidRes(iconRoot, androidResDir)
}

// BuildOptions drives a full Android APK build from an eletrocromo app.
type BuildOptions struct {
	// Config is package identity + go_main (after LoadConfig/Merge/flags).
	Config Config
	// BaseDir resolves relative Config.GoMain (config file directory or cwd).
	BaseDir string
	// WorkDir holds the generated Gradle project. Empty → temp under os.TempDir.
	WorkDir string
	// KeepWorkDir leaves WorkDir in place (always true when WorkDir is set by caller).
	KeepWorkDir bool
	// OutAPK is the destination .apk path (directories created). Required for full build.
	OutAPK string
	// GoOnly stops after multiarch libeletrocromo.so (no Gradle / no SDK).
	GoOnly bool
	// IconRoot is a dist/icons tree with android/mipmap-* (optional; if empty, no mipmaps).
	IconRoot string
	// Stdout/Stderr for subprocess logs (default os.Stdout/Stderr).
	Stdout io.Writer
	Stderr io.Writer
}

// BuildResult is the outcome of Build.
type BuildResult struct {
	// APKPath is the copied/final APK (empty if GoOnly).
	APKPath string
	// WorkDir is the generated Android project path.
	WorkDir string
	// JNILibs lists built native binaries.
	JNILibs []string
}

// Build scaffolds the Android host, cross-compiles the Go app into jniLibs,
// and (unless GoOnly) runs Gradle assembleDebug and copies the APK to OutAPK.
func Build(opts BuildOptions) (*BuildResult, error) {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	cfg, err := opts.Config.withDefaults()
	if err != nil {
		return nil, err
	}
	baseDir := strings.TrimSpace(opts.BaseDir)
	if baseDir == "" {
		baseDir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	goMain, err := ResolveGoMain(cfg.GoMain, baseDir)
	if err != nil {
		return nil, err
	}

	// Stamp version from app tree (git describe) when config omitted version_*.
	vi, name, code := common.StampPackagingVersion(goMain, opts.Config.VersionName, opts.Config.VersionCode)
	cfg.VersionName = name
	cfg.VersionCode = code
	if _, err := fmt.Fprintf(stdout, "eletrocromo: version %s (code %d)\n", cfg.VersionName, cfg.VersionCode); err != nil {
		return nil, err
	}

	workDir, ephemeral, err := common.ResolveWorkDir(opts.WorkDir, "eletrocromo-android-*")
	if err != nil {
		return nil, err
	}
	// Ephemeral work dirs: keep on failure (inspect logs), keep for --go-only
	// or KeepWorkDir; delete after a successful full APK copy.
	var buildErr error
	defer func() {
		if ephemeral && buildErr == nil && !opts.KeepWorkDir && !opts.GoOnly {
			if err := os.RemoveAll(workDir); err != nil && stderr != nil {
				// best-effort cleanup of ephemeral work dir
				fmt.Fprintf(stderr, "eletrocromo: cleanup work dir: %v\n", err)
			}
		}
	}()

	// Host go_main in generated config is absolute so scripts work from workDir.
	genCfg := cfg
	genCfg.GoMain = goMain

	if _, err := fmt.Fprintf(stdout, "eletrocromo: generating Android host in %s\n", workDir); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := Create(Options{
		OutDir: workDir,
		Force:  true, // work dir is ours or full rebuild
		Config: genCfg,
	}); err != nil {
		buildErr = fmt.Errorf("generate host: %w", err)
		return nil, buildErr
	}

	// Manifest expects @mipmap/ic_launcher — always install mipmaps.
	iconRoot := strings.TrimSpace(opts.IconRoot)
	if iconRoot == "" {
		tmpIcons := filepath.Join(workDir, ".eletrocromo-icons")
		if _, err := icons.Generate(icons.Options{OutputDir: tmpIcons, Force: true}); err != nil {
			buildErr = fmt.Errorf("default icons: %w", err)
			return nil, buildErr
		}
		iconRoot = tmpIcons
	}
	resDir := filepath.Join(workDir, "app", "src", "main", "res")
	if _, err := fmt.Fprintf(stdout, "eletrocromo: applying launcher icons from %s\n", iconRoot); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := applyIconMipmaps(iconRoot, resDir); err != nil {
		buildErr = err
		return nil, buildErr
	}

	if _, err := fmt.Fprintf(stdout, "eletrocromo: building Go binary for %v\n", genCfg.abis()); err != nil {
		buildErr = err
		return nil, buildErr
	}
	libs, err := BuildGoLibs(workDir, goMain, genCfg.abis(), vi, genCfg.PackageID, stdout, stderr)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}

	result := &BuildResult{WorkDir: workDir, JNILibs: libs}
	if opts.GoOnly {
		_, err := fmt.Fprintf(stdout, "eletrocromo: --go-only: skipped Gradle (jniLibs ready under %s)\n", workDir)
		return result, err
	}

	outAPK := strings.TrimSpace(opts.OutAPK)
	if outAPK == "" {
		buildErr = ErrOutAPKRequired
		return nil, buildErr
	}
	outAPK, err = filepath.Abs(outAPK)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}

	if _, err := fmt.Fprintf(stdout, "eletrocromo: assembling debug APK…\n"); err != nil {
		buildErr = err
		return nil, buildErr
	}
	apk, err := AssembleDebug(workDir, stdout, stderr)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := os.MkdirAll(filepath.Dir(outAPK), 0o755); err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := copyFile(apk, outAPK); err != nil {
		buildErr = fmt.Errorf("copy apk: %w", err)
		return nil, buildErr
	}
	result.APKPath = outAPK
	_, err = fmt.Fprintf(stdout, "eletrocromo: APK → %s\n", outAPK)
	return result, err
}

// BuildGoLibs cross-compiles the app into workDir/app/src/main/jniLibs/<abi>/libeletrocromo.so.
// stamp is injected via -ldflags -X (goreleaser-style) when apps import internal/version.
func BuildGoLibs(workDir, goMainDir string, abis []string, stamp version.Info, appID string, stdout, stderr io.Writer) ([]string, error) {
	if len(abis) == 0 {
		abis = DefaultABIs
	}
	ldflags := stamp.WithAppID(appID)
	var out []string
	for _, abi := range abis {
		goarch, ok := abiToGOARCH[abi]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnsupportedABI, abi)
		}
		destDir := filepath.Join(workDir, "app", "src", "main", "jniLibs", abi)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return nil, err
		}
		dest := filepath.Join(destDir, "libeletrocromo.so")
		if _, err := fmt.Fprintf(stdout, "  → %s (GOARCH=%s)\n", abi, goarch); err != nil {
			return nil, err
		}
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", dest, ".")
		cmd.Dir = goMainDir
		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS=android",
			"GOARCH="+goarch,
		)
		if goarch == "arm" {
			cmd.Env = append(cmd.Env, "GOARM=7")
		}
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("go build %s (GOARCH=%s CGO_ENABLED=0): %w\nnote: pure Go android builds typically only support arm64-v8a without an NDK; set abis in eletrocromo.json", abi, goarch, err)
		}
		out = append(out, dest)
	}
	return out, nil
}

// AssembleDebug runs Gradle assembleDebug in workDir and returns the debug APK path.
func AssembleDebug(workDir string, stdout, stderr io.Writer) (string, error) {
	if err := requireJDK(); err != nil {
		return "", err
	}
	sdk, err := androidSDK()
	if err != nil {
		return "", err
	}
	if err := writeLocalProperties(workDir, sdk); err != nil {
		return "", err
	}

	gradle, err := resolveGradle(workDir)
	if err != nil {
		return "", err
	}
	args := append(gradle[1:], "assembleDebug", "--stacktrace")
	cmd := exec.Command(gradle[0], args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(),
		"ANDROID_HOME="+sdk,
		"ANDROID_SDK_ROOT="+sdk,
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gradle assembleDebug: %w\n(work dir left at %s)", err, workDir)
	}

	// Standard AGP debug output.
	candidates := []string{
		filepath.Join(workDir, "app", "build", "outputs", "apk", "debug", "app-debug.apk"),
	}
	matches, err := filepath.Glob(filepath.Join(workDir, "app", "build", "outputs", "apk", "debug", "*.apk"))
	if err != nil {
		return "", fmt.Errorf("glob debug apk: %w", err)
	}
	candidates = append(candidates, matches...)
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p, nil
		}
	}
	return "", fmt.Errorf("%w (work dir %s)", ErrDebugAPKMissing, workDir)
}

func requireJDK() error {
	if _, err := exec.LookPath("java"); err != nil {
		return fmt.Errorf("java not found on PATH (need JDK 17+ for Gradle): %w", err)
	}
	return nil
}

func androidSDK() (string, error) {
	for _, key := range []string{"ANDROID_HOME", "ANDROID_SDK_ROOT"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			if st, err := os.Stat(v); err == nil && st.IsDir() {
				return v, nil
			}
			return "", fmt.Errorf("%w: %s=%q", ErrSDKEnvNotDir, key, v)
		}
	}
	// Common user installs.
	home, err := os.UserHomeDir()
	if err == nil {
		for _, rel := range []string{
			"Android/Sdk",
			"Library/Android/sdk",
			"AppData/Local/Android/Sdk",
		} {
			p := filepath.Join(home, rel)
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				return p, nil
			}
		}
	}
	return "", ErrAndroidSDKNotFound
}

func writeLocalProperties(workDir, sdk string) error {
	// Gradle local.properties wants forward slashes / escaped backslashes.
	sdkProp := filepath.ToSlash(sdk)
	body := fmt.Sprintf("## Generated by eletrocromo android build\nsdk.dir=%s\n", sdkProp)
	return os.WriteFile(filepath.Join(workDir, "local.properties"), []byte(body), 0o644)
}

func resolveGradle(workDir string) ([]string, error) {
	wrapper := filepath.Join(workDir, "gradlew")
	if st, err := os.Stat(wrapper); err == nil && !st.IsDir() {
		return []string{wrapper}, nil
	}
	// Bootstrap wrapper if system gradle exists.
	if g, err := exec.LookPath("gradle"); err == nil {
		// Prefer system gradle directly (no wrapper jar in template).
		return []string{g}, nil
	}
	return nil, ErrGradleNotFound
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return errors.Join(err, in.Close())
	}
	_, copyErr := io.Copy(out, in)
	// Join so a close failure is not dropped when Copy already failed.
	return errors.Join(copyErr, out.Close(), in.Close())
}

// DefaultOutAPK suggests dist/<last-label>-debug.apk under cwd.
func DefaultOutAPK(packageID, cwd string) string {
	label := packageID
	if i := strings.LastIndex(packageID, "."); i >= 0 {
		label = packageID[i+1:]
	}
	if label == "" {
		label = "app"
	}
	return filepath.Join(cwd, "dist", label+"-debug.apk")
}
