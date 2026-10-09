package build

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/workflow"
	"github.com/stretchr/testify/require"
)

func TestWorkflowSteps(t *testing.T) {
	dir := t.TempDir()
	host := Host{
		Spec:   Spec{Dir: dir, ID: "br.tec.lew.demo", Name: "Demo"},
		GOARCH: "arm64",
		Out:    filepath.Join(dir, "Demo_darwin_arm64.app"),
	}
	graph, err := host.Workflow("darwin")
	require.NoError(t, err)
	require.Equal(t, []string{"darwin/arm64"}, graph.Defaults)
	require.Equal(t, []string{"icons", "scaffold", "darwin/arm64"}, stepNames(graph))
	require.Equal(t, []string{"icons"}, graph.Steps[1].Deps)
	require.Equal(t, []string{"scaffold"}, graph.Steps[2].Deps)

	graph, err = host.Workflow("linux")
	require.NoError(t, err)
	require.Equal(t, []string{"linux/arm64"}, graph.Defaults)
	require.Equal(t, []string{"icons", "linux/arm64"}, stepNames(graph))
	require.Equal(t, []string{"icons"}, graph.Steps[1].Deps)

	host.IconRoot = filepath.Join(dir, "icons")
	graph, err = host.Workflow("ios")
	require.NoError(t, err)
	require.Equal(t, []string{"scaffold", "ios/arm64"}, stepNames(graph))
	require.Equal(t, []string{"scaffold"}, graph.Steps[1].Deps)

	_, err = host.Workflow("plan9")
	require.EqualError(t, err, "plan9 has no app package")
}

func TestWorkflowScaffoldsMacHost(t *testing.T) {
	mainDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	work := t.TempDir()
	host := Host{
		Spec:   Spec{Dir: mainDir, ID: "br.tec.lew.drivers", Name: "Drivers"},
		Work:   work,
		GOARCH: "arm64",
		Out:    filepath.Join(t.TempDir(), "Drivers_darwin_arm64.app"),
	}
	graph, err := host.Workflow("darwin")
	require.NoError(t, err)
	_, err = workflow.Run(t.Context(), graph, []string{"scaffold"})
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(work, "Sources", "AppDelegate.swift"))
	require.NoError(t, err)
	window, err := os.ReadFile(filepath.Join(work, "Sources", "MainWindow.swift"))
	require.NoError(t, err)
	require.Contains(t, string(window), "makeKeyAndOrderFront")
	server, err := os.ReadFile(filepath.Join(work, "Sources", "ServerProcess.swift"))
	require.NoError(t, err)
	require.Contains(t, string(server), `helperName = "eletrocromo-server"`)
	body, err := os.ReadFile(filepath.Join(work, "project.yml"))
	require.NoError(t, err)
	require.Contains(t, string(body), "Drivers")
	plist, err := os.ReadFile(filepath.Join(work, "Info.plist"))
	require.NoError(t, err)
	require.Contains(t, string(plist), "<key>CFBundleExecutable</key>\n\t<string>Drivers</string>")
}

func TestWorkflowPackagesWindowsExe(t *testing.T) {
	mainDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	out := filepath.Join(t.TempDir(), "Demo_windows_amd64.exe")
	host := Host{
		Spec:   Spec{Dir: mainDir, ID: "br.tec.lew.demo", Name: "Demo", Version: "1.2.3"},
		Out:    out,
		GOARCH: "amd64",
	}
	graph, err := host.Workflow("windows")
	require.NoError(t, err)
	_, err = workflow.Run(t.Context(), graph, nil)
	require.NoError(t, err)
	body, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Contains(t, string(body), "IHDR")
}

func TestWorkflowPackagesLinuxAppImage(t *testing.T) {
	mainDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	out := filepath.Join(t.TempDir(), "Demo_linux_amd64.AppImage")
	host := Host{
		Spec:   Spec{Dir: mainDir, ID: "br.tec.lew.demo", Name: "Demo", Version: "1.2.3"},
		Out:    out,
		GOARCH: "amd64",
	}
	graph, err := host.Workflow("linux")
	require.NoError(t, err)
	_, err = workflow.Run(t.Context(), graph, nil)
	require.NoError(t, err)
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, "\x7fELF", string(raw[:4]))
	tr, err := release.OpenTrailer(out)
	require.NoError(t, err)
	defer tr.Close()
	icon, err := tr.Bytes("icon.png")
	require.NoError(t, err)
	require.Contains(t, string(icon), "IHDR")
	marker, err := tr.Bytes(release.MarkerFile)
	require.NoError(t, err)
	require.Equal(t, "br.tec.lew.demo\n", string(marker))
}

func stepNames(graph workflow.Graph) []string {
	names := make([]string, len(graph.Steps))
	for i, step := range graph.Steps {
		names[i] = step.Name
	}
	return names
}
