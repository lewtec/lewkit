package tool

import (
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallArtifactUnzipsAndStripsTopDirectory(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "demo.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("demo-1.0.0/bin/demo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("payload")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.ServeFile(writer, request, archivePath)
	}))
	t.Cleanup(server.Close)

	destination := filepath.Join(t.TempDir(), "out")
	artifact := Artifact{URL: server.URL + "/demo.zip", OS: "linux", Arch: "amd64"}
	if err := InstallArtifact(t.Context(), artifact, destination, DownloadOptions{}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(destination, "bin", "demo"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "payload" {
		t.Fatalf("body = %q", body)
	}
}

func TestDownloadFileRejectsHashMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write([]byte("hello"))
	}))
	t.Cleanup(server.Close)
	destination := filepath.Join(t.TempDir(), "copy.txt")
	err := DownloadFile(t.Context(), server.URL, destination, DownloadOptions{Hash: "sha256:deadbeef"})
	if err == nil {
		t.Fatal("expected hash mismatch")
	}
}
