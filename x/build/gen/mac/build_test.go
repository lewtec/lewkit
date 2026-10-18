package mac

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/exec/native"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestInstallHelper(t *testing.T) {
	app := filepath.Join(t.TempDir(), "Counter.app")
	macOS := filepath.Join(app, "Contents", "MacOS")
	require.NoError(t, os.MkdirAll(macOS, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(macOS, "Counter"), []byte("swift-shell"), 0o755))
	helper := filepath.Join(t.TempDir(), HelperName)
	require.NoError(t, os.WriteFile(helper, []byte("go-server"), 0o644))
	require.NoError(t, installHelper(app, helper))
	body, err := os.ReadFile(filepath.Join(macOS, HelperName))
	require.NoError(t, err)
	if string(body) != "go-server" {
		require.FailNowf(t, "", "server %q", body)
	}
	shell, err := os.ReadFile(filepath.Join(macOS, "Counter"))
	require.NoError(t, err)
	if string(shell) != "swift-shell" {
		require.FailNowf(t, "", "shell %q", shell)
	}
	st, err := os.Stat(filepath.Join(macOS, HelperName))
	require.NoError(t, err)
	if st.Mode().Perm()&0o111 == 0 {
		require.FailNowf(t, "", "server mode %v", st.Mode())
	}
}

func TestMacXcodeArgs(t *testing.T) {
	arm, err := darwinXArch("arm64")
	require.NoError(t, err)
	got := macXcodeArgs("Demo.xcodeproj", "Demo", arm, "Derived")
	require.Equal(t, []string{
		"-project", "Demo.xcodeproj",
		"-scheme", "Demo",
		"-configuration", "Debug",
		"-sdk", "macosx",
		"-destination", "generic/platform=macOS",
		"-derivedDataPath", "Derived",
		"-skipPackagePluginValidation",
		"CODE_SIGN_STYLE=Manual",
		"CODE_SIGN_IDENTITY=-",
		"CODE_SIGNING_REQUIRED=NO",
		"CODE_SIGNING_ALLOWED=YES",
		"AD_HOC_CODE_SIGNING_ALLOWED=YES",
		"ENABLE_DEBUG_DYLIB=NO",
		"ARCHS=arm64",
		"ONLY_ACTIVE_ARCH=NO",
		"build",
	}, got)

	intel, err := darwinXArch("amd64")
	require.NoError(t, err)
	intelArgs := macXcodeArgs("Demo.xcodeproj", "Demo", intel, "Derived")
	require.Contains(t, intelArgs, "ARCHS=x86_64")
	require.NotContains(t, intelArgs, "ONLY_ACTIVE_ARCH=YES")
	require.NotContains(t, intelArgs, "DEVELOPMENT_TEAM=")
}

func TestAssembleMacReportsXcodebuildDiagnostic(t *testing.T) {
	if runtime.GOOS == "windows" {
		test.Skip(t, "fake xcodebuild is a shell script")
	}
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "xcodegen"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	script := "#!/bin/sh\nprintf '%s\\n' 'note: compiling'\nprintf '%s\\n' 'error: Signing for Demo requires a development team.'\nexit 65\n"
	require.NoError(t, os.WriteFile(filepath.Join(bin, "xcodebuild"), []byte(script), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	work := t.TempDir()

	test.Stderr(t, func() {
		_, err := assembleMac(t.Context(), work, "Demo", "arm64")
		require.ErrorIs(t, err, ErrXcodebuild)
		require.ErrorIs(t, err, ErrWorkDirLeftAt)
		require.Contains(t, err.Error(), "error: Signing for Demo requires a development team.")
		require.Contains(t, err.Error(), work)
		require.NotContains(t, err.Error(), "note: compiling")
	})
}

func TestBuildFullAppNeedsMacOS(t *testing.T) {
	if runtime.GOOS == "darwin" {
		test.Skip(t, "full build uses xcodebuild")
	}
	mainDir := t.TempDir()
	_, err := Build(t.Context(), BuildOptions{
		Config: Config{
			PackageID:   "br.tec.lew.demo",
			AppName:     "Demo",
			VersionName: "1.2.3",
			VersionCode: 5,
			GoMain:      mainDir,
		},
		BaseDir: mainDir,
		WorkDir: t.TempDir(),
		OutApp:  filepath.Join(t.TempDir(), "Demo.app"),
	})
	require.ErrorIs(t, err, ErrMacOSRequired)
}
