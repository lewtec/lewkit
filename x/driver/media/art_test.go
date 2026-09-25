package media

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestArtCacheLocal(t *testing.T) {
	got, err := GetArtCachePath(context.Background(), "file:///tmp/cover.png")
	if err != nil || got != "/tmp/cover.png" {
		t.Fatalf("file: %q %v", got, err)
	}
	got, err = GetArtCachePath(context.Background(), "audio-x-generic")
	if err != nil || got != "audio-x-generic" {
		t.Fatalf("plain: %q %v", got, err)
	}
}

func TestArtCacheHTTP(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte("png"))
	}))
	t.Cleanup(srv.Close)
	first, err := GetArtCachePath(context.Background(), srv.URL+"/cover")
	if err != nil {
		t.Fatal(err)
	}
	second, err := GetArtCachePath(context.Background(), srv.URL+"/cover")
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
