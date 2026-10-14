package mac

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
