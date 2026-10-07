package media

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/lewtec/lewkit/x/driver/dirs/os"
	_ "github.com/lewtec/lewkit/x/driver/httpclient/native"
)

func TestArtCacheLocal(t *testing.T) {
	got, err := GetArtCachePath(t.Context(), "file:///tmp/cover.png")
	if err != nil || got != "/tmp/cover.png" {
		t.Fatalf("file: %q %v", got, err)
	}
	got, err = GetArtCachePath(t.Context(), "audio-x-generic")
	if err != nil || got != "audio-x-generic" {
		t.Fatalf("plain: %q %v", got, err)
	}
}

func TestArtCacheHTTP(t *testing.T) {
	root := t.TempDir()
	t.Setenv("LEWKIT_DATA_DIR", filepath.Join(root, "data"))
	t.Setenv("LEWKIT_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("LEWKIT_CONFIG_DIR", filepath.Join(root, "config"))
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("png"))
	}))
	t.Cleanup(srv.Close)
	first, err := GetArtCachePath(t.Context(), srv.URL+"/cover")
	if err != nil {
		t.Fatal(err)
	}
	second, err := GetArtCachePath(t.Context(), srv.URL+"/cover")
	if err != nil {
		t.Fatal(err)
	}
	if first != second || hits != 1 {
		t.Fatalf("path %q hits %d", first, hits)
	}
	body, err := os.ReadFile(first)
	if err != nil || string(body) != "png" {
		t.Fatalf("body %q %v", body, err)
	}
	if filepath.Base(filepath.Dir(first)) != "media-art" {
		t.Fatalf("dir %s", first)
	}
}
