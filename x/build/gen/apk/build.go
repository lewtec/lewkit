package apk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/lewtec/lewkit/x/build/version"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Sentinel errors for static build/setup failures (errors.Is / wrap with %w).
var (
	ErrOutAPKRequired     = errors.New("out apk path is required (or use --go-only)")
	ErrUnsupportedABI     = errors.New("unsupported abi")
	ErrDebugAPKMissing    = errors.New("gradle succeeded but no debug APK under app/build/outputs/apk/debug")
	ErrSDKEnvNotDir       = errors.New("android SDK env path is not a directory")
	ErrAndroidSDKNotFound = errors.New("android SDK not found: set ANDROID_HOME or install mise android-sdk@13.0")
	ErrGradleNotFound     = errors.New("neither ./gradlew nor gradle on PATH; install Gradle 8.9+ or mise gradle@9.6.1")
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
	// CGO builds the JNI library with cgo and the NDK clang for each ABI.
	CGO bool
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
func Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	stdout := opts.Stdout
	if stdout == nil {
		stdout = taskgroup.LineWriterFrom(ctx)
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = stdout
	}

	cfg, err := opts.Config.withDefaults(ctx)
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
	vi, name, code := common.StampPackagingVersion(ctx, goMain, opts.Config.VersionName, opts.Config.VersionCode)
	cfg.VersionName = name
	cfg.VersionCode = code
	slog.Info("android version", "version", cfg.VersionName, "code", cfg.VersionCode)

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
				slog.Warn("android cleanup", "err", err)
			}
		}
	}()

	// Host go_main in generated config is absolute so scripts work from workDir.
	genCfg := cfg
	genCfg.GoMain = goMain

	slog.Info("android host", "dir", workDir)
	if err := Create(ctx, Options{
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
		if _, err := icons.Generate(icons.Options{OutputDir: tmpIcons, Force: true, SourcePath: icons.MasterPath(baseDir, cfg.Icon)}); err != nil {
			buildErr = fmt.Errorf("icons: %w", err)
			return nil, buildErr
		}
		iconRoot = tmpIcons
	}
	resDir := filepath.Join(workDir, "app", "src", "main", "res")
	slog.Info("android icons", "dir", iconRoot)
	if err := applyIconMipmaps(iconRoot, resDir); err != nil {
		buildErr = err
		return nil, buildErr
	}

	slog.Info("android go", "abis", genCfg.abis())
	libs, err := BuildGoLibs(ctx, workDir, goMain, genCfg.abis(), vi, genCfg.PackageID, opts.CGO, stdout)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}

	result := &BuildResult{WorkDir: workDir, JNILibs: libs}
	if opts.GoOnly {
		slog.Info("android go-only", "dir", workDir)
		return result, nil
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

	slog.Info("android gradle")
	apk, err := AssembleDebug(ctx, workDir)
	if err != nil {
		buildErr = err
		return nil, buildErr
	}
	if err := common.CopyFile(apk, outAPK, 0o644); err != nil {
		buildErr = fmt.Errorf("copy apk: %w", err)
		return nil, buildErr
	}
	result.APKPath = outAPK
	slog.Info("android apk", "path", outAPK)
	return result, nil
}

// BuildGoLibs cross-compiles the app into workDir/app/src/main/jniLibs/<abi>/libeletrocromo.so.
// stamp is injected via -ldflags -X (goreleaser-style) when apps import internal/version.
func BuildGoLibs(ctx context.Context, workDir, goMainDir string, abis []string, stamp version.Info, appID string, cgoEnabled bool, stdout io.Writer) ([]string, error) {
	if len(abis) == 0 {
		abis = DefaultABIs
	}
	ldflags := stamp.WithAppID(appID)
	toolexec, err := nocgoAndroidToolexec(ctx, abis, cgoEnabled)
	if err != nil {
		return nil, err
	}
	if toolexec != "" {
		defer os.RemoveAll(filepath.Dir(toolexec))
	}
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
		slog.Info("android abi", "abi", abi, "goarch", goarch)
		cgoFlag := "0"
		linkFlags := ldflags
		args := []string{"-trimpath", "-ldflags", linkFlags, "-o", dest, "."}
		if cgoEnabled {
			args = []string{"-buildmode=c-shared", "-trimpath", "-ldflags", linkFlags, "-o", dest, "."}
		} else if goarch == "arm64" {
			// JNI_OnLoad and Hook.call are published only with these three.
			linkFlags = strings.TrimSpace(ldflags + " -checklinkname=0")
			args = []string{
				"-trimpath",
				"-tags", "androidnocgo",
				"-toolexec", toolexec,
				"-ldflags", linkFlags,
				"-o", dest, ".",
			}
		}
		env := append(os.Environ(), "GOOS=android", "GOARCH="+goarch)
		if goarch == "arm" {
			env = append(env, "GOARM=7")
		}
		if cgoEnabled {
			cc, err := ndkCC(ctx, goarch)
			if err != nil {
				return nil, err
			}
			cgoFlag = "1"
			env = append(env, "CC="+cc)
			slog.Info("android cgo", "cc", cc)
		}
		env = append(env, "CGO_ENABLED="+cgoFlag)
		if err := (gocmd.Command{Dir: goMainDir, Env: env, Args: args}).Run(ctx); err != nil {
			return nil, fmt.Errorf("go build %s (GOARCH=%s CGO_ENABLED=%s): %w", abi, goarch, cgoFlag, err)
		}
		out = append(out, dest)
	}
	return out, nil
}

// nocgoAndroidToolexec builds the host wrapper that publishes JNI_OnLoad
// for a cgo-free arm64 library. Other ABIs do not use it.
func nocgoAndroidToolexec(ctx context.Context, abis []string, cgoEnabled bool) (string, error) {
	if cgoEnabled {
		return "", nil
	}
	need := false
	for _, abi := range abis {
		if abiToGOARCH[abi] == "arm64" {
			need = true
			break
		}
	}
	if !need {
		return "", nil
	}
	dir, err := os.MkdirTemp("", "lewkit-androidtoolexec-")
	if err != nil {
		return "", err
	}
	dest := filepath.Join(dir, "androidtoolexec")
	err = (gocmd.Command{
		Args: []string{"-o", dest, "github.com/lewtec/lewkit/x/build/androidtoolexec"},
	}).Run(ctx)
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("android toolexec: %w", err)
	}
	return dest, nil
}

// AssembleDebug runs Gradle assembleDebug in workDir and returns the debug APK path.
func AssembleDebug(ctx context.Context, workDir string) (string, error) {
	home, err := javaHome(ctx)
	if err != nil {
		return "", err
	}
	sdk, err := androidSDK(ctx)
	if err != nil {
		return "", err
	}
	if err := writeLocalProperties(workDir, sdk); err != nil {
		return "", err
	}

	gradle, err := resolveGradle(ctx, workDir)
	if err != nil {
		return "", err
	}
	args := append(gradle[1:], "assembleDebug", "--stacktrace")
	env := gradleEnv(os.Environ(), home, sdk)
	if err := gocmd.Tool(ctx, gradle[0], workDir, env, args...); err != nil {
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

func writeLocalProperties(workDir, sdk string) error {
	// Gradle local.properties wants forward slashes / escaped backslashes.
	sdkProp := filepath.ToSlash(sdk)
	body := fmt.Sprintf("## Generated by eletrocromo android build\nsdk.dir=%s\n", sdkProp)
	return os.WriteFile(filepath.Join(workDir, "local.properties"), []byte(body), 0o644)
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
