package tool

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/require"
)

func TestInstallArtifactUnzipsAndStripsTopDirectory(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "demo.zip")
	file, err := os.Create(archivePath)
	require.NoError(t, err)
	test.CloseOnCleanup(t, file)
	writer := zip.NewWriter(file)
	entry, err := writer.Create("demo-1.0.0/bin/demo")
	require.NoError(t, err)
	_, err = entry.Write([]byte("payload"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, archivePath)
	}))
	t.Cleanup(server.Close)

	destination := filepath.Join(t.TempDir(), "out")
	artifact := Artifact{URL: server.URL + "/demo.zip", OS: "linux", Arch: "amd64"}
	require.NoError(t, InstallArtifact(t.Context(), artifact, destination, DownloadOptions{}))
	body, err := os.ReadFile(filepath.Join(destination, "bin", "demo"))
	require.NoError(t, err)
	require.Equal(t, "payload", string(body))
}

func TestDownloadFileRejectsHashMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte("hello"))
	}))
	t.Cleanup(server.Close)
	destination := filepath.Join(t.TempDir(), "copy.txt")
	err := DownloadFile(t.Context(), server.URL, destination, DownloadOptions{Hash: "sha256:deadbeef"})
	require.Error(t, err)
}
