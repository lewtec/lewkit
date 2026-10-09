package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLaunchLinuxApp(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux app")
	}
	app := filepath.Join(t.TempDir(), "Demo.app")
	require.NoError(t, os.MkdirAll(app, 0o755))
	script := "#!/bin/sh\nexit 0\n"
	require.NoError(t, os.WriteFile(filepath.Join(app, "AppRun"), []byte(script), 0o755))
	require.NoError(t, launchApp(t.Context(), "linux", app, "br.tec.lew.demo"))
}
