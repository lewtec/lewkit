package ios

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/exec/native"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestIOSXcodeArgs(t *testing.T) {
	dest, products := xcodeDestination(string(SDKSimulator))
	require.Equal(t, "Debug-iphonesimulator", products)
	got := iosXcodeArgs("Demo.xcodeproj", "Demo", string(SDKSimulator), dest, "arm64", "Derived")
	require.Contains(t, got, "generic/platform=iOS Simulator")
	require.Contains(t, got, "CODE_SIGN_STYLE=Manual")
	require.Contains(t, got, "ARCHS=arm64")
	require.Contains(t, got, "ONLY_ACTIVE_ARCH=NO")
	require.Contains(t, got, "EXCLUDED_ARCHS=x86_64")
	require.NotContains(t, got, "ONLY_ACTIVE_ARCH=YES")

	deviceDest, deviceProducts := xcodeDestination(string(SDKDevice))
	require.Equal(t, "Debug-iphoneos", deviceProducts)
	device := iosXcodeArgs("Demo.xcodeproj", "Demo", string(SDKDevice), deviceDest, "x86_64", "Derived")
	require.Contains(t, device, "generic/platform=iOS")
	require.Contains(t, device, "ARCHS=x86_64")
	require.Contains(t, device, "EXCLUDED_ARCHS=arm64")
}

func TestAssembleDebugReportsXcodebuildDiagnostic(t *testing.T) {
	if runtime.GOOS == "windows" {
		test.Skip(t, "fake xcodebuild is a shell script")
	}
	bin := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bin, "xcodegen"), []byte("#!/bin/sh\nexit 0\n"), 0o755))
	script := "#!/bin/sh\nprintf '%s\\n' 'error: No signing certificate \"iOS Development\" found.'\nexit 65\n"
	require.NoError(t, os.WriteFile(filepath.Join(bin, "xcodebuild"), []byte(script), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	work := t.TempDir()

	test.Stderr(t, func() {
		_, err := assembleDebug(t.Context(), work, "Demo", string(SDKSimulator), "arm64")
		require.ErrorIs(t, err, ErrXcodebuild)
		require.ErrorIs(t, err, ErrWorkDirLeftAt)
		require.Contains(t, err.Error(), `error: No signing certificate "iOS Development" found.`)
		require.Contains(t, err.Error(), work)
	})
}
