package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/exec/native"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestXcodeDiagnosticPrefersErrorLines(t *testing.T) {
	log := &xcodeLog{}
	_, err := log.Write([]byte("note: compiling\nerror: Signing for Demo requires a development team.\n** BUILD FAILED **\n"))
	require.NoError(t, err)
	require.Equal(t, "error: Signing for Demo requires a development team.", log.diagnostic())
}

func TestXcodeDiagnosticKeepsErrorTail(t *testing.T) {
	log := &xcodeLog{}
	var buf strings.Builder
	for i := range xcodeDiagnosticLines + 10 {
		fmt.Fprintf(&buf, "error: step %d\n", i)
	}
	_, err := log.Write([]byte(buf.String()))
	require.NoError(t, err)
	got := strings.Split(log.diagnostic(), "\n")
	require.Len(t, got, xcodeDiagnosticLines)
	require.Equal(t, fmt.Sprintf("error: step %d", 10), got[0])
	require.Equal(t, fmt.Sprintf("error: step %d", xcodeDiagnosticLines+9), got[len(got)-1])
}

func TestXcodeDiagnosticFallsBackToTail(t *testing.T) {
	log := &xcodeLog{}
	var buf strings.Builder
	for i := range xcodeDiagnosticLines + 5 {
		fmt.Fprintf(&buf, "line %d\n", i)
	}
	_, err := log.Write([]byte(buf.String()))
	require.NoError(t, err)
	got := strings.Split(log.diagnostic(), "\n")
	require.Len(t, got, xcodeDiagnosticLines)
	require.Equal(t, "line 5", got[0])
	require.Equal(t, fmt.Sprintf("line %d", xcodeDiagnosticLines+4), got[len(got)-1])
}

func TestRunXcodebuildIncludesDiagnostics(t *testing.T) {
	if runtime.GOOS == "windows" {
		test.Skip(t, "fake xcodebuild is a shell script")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\\n' 'note: compiling'\nprintf '%s\\n' 'error: Signing for Demo requires a development team.'\nexit 65\n"
	require.NoError(t, os.WriteFile(filepath.Join(bin, "xcodebuild"), []byte(script), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	logged := test.Stderr(t, func() {
		err := RunXcodebuild(t.Context(), t.TempDir(), "-project", "Demo.xcodeproj", "build")
		require.ErrorIs(t, err, ErrXcodebuild)
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		require.Contains(t, err.Error(), "error: Signing for Demo requires a development team.")
		require.NotContains(t, err.Error(), "note: compiling")
	})
	require.Contains(t, logged, "error: Signing for Demo requires a development team.")
}
