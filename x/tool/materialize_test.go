package tool

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestDownloadFileSingleflightsURL(t *testing.T) {
	var hits atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if hits.Add(1) == 1 {
			close(entered)
			<-release
		}
		_, _ = writer.Write([]byte("payload"))
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	fetch := func(name string) {
		defer wg.Done()
		destination := filepath.Join(dir, name)
		errs <- DownloadFile(t.Context(), server.URL, destination, DownloadOptions{})
	}
	wg.Add(1)
	go fetch("a.txt")
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not start")
	}
	wg.Add(1)
	go fetch("b.txt")
	require.Eventually(t, func() bool {
		return downloadRefs(server.URL, "") == 2
	}, 5*time.Second, 5*time.Millisecond)
	require.Equal(t, int32(1), hits.Load())
	close(release)
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		body, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		require.Equal(t, "payload", string(body))
	}
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
