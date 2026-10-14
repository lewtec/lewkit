package apk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/icons"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrAndroidEntryDir               = errors.New("android entry dir")
	ErrAndroidEntryDirIsEmpty        = errs.Wrap(errs.ErrEmpty, "android entry dir is empty")
	ErrAndroidToolexec               = errors.New("android toolexec")
	ErrBuildABICgoEnabled            = errors.New("CGO_ENABLED")
	ErrEmptyToolexecArg              = errs.Wrap(errs.ErrEmpty, "empty toolexec arg")
	ErrGlobDebugApk                  = errors.New("glob debug apk")
	ErrGradleAssembledebug           = errors.New("gradle assembleDebug")
	ErrLoadConfig                    = errors.New("")
	ErrNormalizeConfigPackageId      = errors.New("package id")
	ErrToolexecArgContainsBothQuotes = errors.New("toolexec arg contains both quotes")
	ErrToolexecIsEmpty               = errs.Wrap(errs.ErrEmpty, "toolexec is empty")
)

// Sentinel errors for static build/setup failures (errors.Is / wrap with %w).
var (
	ErrOutAPKRequired     = errs.Wrap(errs.ErrRequired, "out apk path is required (or use --go-only)")
	ErrUnsupportedABI     = errs.Wrap(errs.ErrUnsupported, "unsupported abi")
	ErrDebugAPKMissing    = errs.Wrap(errs.ErrMissing, "gradle succeeded but no debug APK under app/build/outputs/apk/debug")
	ErrSDKEnvNotDir       = errs.Wrap(errs.ErrInvalid, "android SDK env path is not a directory")
	ErrAndroidSDKNotFound = errs.Wrap(errs.ErrNotFound, "android SDK not found: set ANDROID_HOME or install mise android-sdk@13.0")
	ErrGradleNotFound     = errs.Wrap(errs.ErrNotFound, "neither ./gradlew nor gradle on PATH; install Gradle 8.9+ or mise gradle@9.6.1")
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
	// SkipGenerate leaves an existing scaffold in WorkDir in place.
	SkipGenerate bool
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
	stdout, stderr := common.Writers(ctx, opts.Stdout, opts.Stderr)

	cfg, err := opts.Config.withDefaults(ctx)
	if err != nil {
		return nil, err
	}
	baseDir, err := common.WorkingDir(opts.BaseDir)
	if err != nil {
		return nil, err
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
		common.CleanupWorkDir(ephemeral, buildErr, opts.KeepWorkDir, opts.GoOnly, workDir, stderr, "android")
	}()

	// Host go_main in generated config is absolute so scripts work from workDir.
	genCfg := cfg
	genCfg.GoMain = goMain

	if !opts.SkipGenerate {
		slog.Info("android host", "dir", workDir)
		if err := Create(ctx, Options{
			OutDir: workDir,
			Force:  true, // work dir is ours or full rebuild
			Config: genCfg,
		}); err != nil {
			buildErr = fmt.Errorf("generate host: %w", err)
			return nil, buildErr
		}
	}

	// Manifest expects @mipmap/ic_launcher — always install mipmaps.
	iconRoot, err := common.IconRoot(baseDir, cfg.Icon, opts.IconRoot, workDir, ".eletrocromo-icons")
	if err != nil {
		buildErr = err
		return nil, buildErr
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

// nocgoAndroidToolexec builds the host wrapper that publishes JNI_OnLoad
// for a cgo-free arm64 library. Other ABIs do not use it.
// toolDir is removed by the caller. arg is the -toolexec value: the wrapper
// and the export file. An empty arg means this build does not use the wrapper.
func nocgoAndroidToolexec(ctx context.Context, abis []string, cgoEnabled bool) (toolDir, arg string, err error) {
	if cgoEnabled {
		return "", "", nil
	}
	need := false
	for _, abi := range abis {
		if abiToGOARCH[abi] == "arm64" {
			need = true
			break
		}
	}
	if !need {
		return "", "", nil
	}
	export, err := entryExportFile(ctx)
	if err != nil {
		return "", "", err
	}
	dir, err := os.MkdirTemp("", "lewkit-androidtoolexec-")
	if err != nil {
		return "", "", err
	}
	dest := filepath.Join(dir, "androidtoolexec")
	// The caller is an orchestrator. This host binary is in the embedded
	// compiler's scope: no cgo, no -buildmode, and no -toolexec.
	err = (gocmd.Command{
		Dir:  lewkitDir(),
		Args: []string{"-o", dest, "github.com/lewtec/lewkit/x/build/androidtoolexec"},
	}).Run(ctx)
	if err != nil {
		os.RemoveAll(dir)
		return "", "", fmt.Errorf("%w: %w", ErrAndroidToolexec, err)
	}
	joined, err := joinToolexec(dest, export)
	if err != nil {
		os.RemoveAll(dir)
		return "", "", err
	}
	return dir, joined, nil
}

func entryExportFile(ctx context.Context) (string, error) {
	cmd := execdriver.MustCommand(ctx, "go", "list", "-f", "{{.Dir}}", "github.com/lewtec/lewkit/x/entry/android")
	// The app main may be another module. go list has to run in this one.
	cmd.Dir = lewkitDir()
	out, err := execdriver.Output(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrAndroidEntryDir, err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", ErrAndroidEntryDirIsEmpty
	}
	file := filepath.Join(dir, "_cgo_android_export.go")
	if _, err := os.Stat(file); err != nil {
		return "", err
	}
	return file, nil
}

// lewkitDir is the directory of the module that contains this package.
// An empty result leaves go in the process working directory.
func lewkitDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok || !filepath.IsAbs(file) {
		return ""
	}
	dir := filepath.Dir(file)
	for {
		body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && strings.Contains(string(body), "module github.com/lewtec/lewkit") {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// joinToolexec is the -toolexec value. cmd/go splits it on spaces the way
// a shell would, so a path with a space is quoted.
func joinToolexec(bin, export string) (string, error) {
	b, err := quoteToolexec(bin)
	if err != nil {
		return "", err
	}
	e, err := quoteToolexec(export)
	if err != nil {
		return "", err
	}
	return b + " " + e, nil
}

func quoteToolexec(s string) (string, error) {
	if s == "" {
		return "", ErrEmptyToolexecArg
	}
	if !strings.ContainsAny(s, " \t\n\r'\"") {
		return s, nil
	}
	if strings.Contains(s, "'") && strings.Contains(s, `"`) {
		return "", fmt.Errorf("%w: %s", ErrToolexecArgContainsBothQuotes, s)
	}
	if strings.Contains(s, "'") {
		return `"` + s + `"`, nil
	}
	return "'" + s + "'", nil
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
		return "", fmt.Errorf("%w: %w\n(work dir left at %s)", ErrGradleAssembledebug, err, workDir)
	}

	// Standard AGP debug output.
	candidates := []string{
		filepath.Join(workDir, "app", "build", "outputs", "apk", "debug", "app-debug.apk"),
	}
	matches, err := filepath.Glob(filepath.Join(workDir, "app", "build", "outputs", "apk", "debug", "*.apk"))
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrGlobDebugApk, err)
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
