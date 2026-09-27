package main

import (
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	lewimage "github.com/lewtec/lewkit/x/image"
)

func TestHomeListsADriver(t *testing.T) {
	srv := httptest.NewServer(newPage(t.Context()))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "Open triangle") {
		t.Fatal("missing button")
	}
	if !strings.Contains(string(body), "<h2>") {
		t.Fatal("missing driver section")
	}
	if strings.Contains(string(body), "location.href") {
		t.Fatal("triangle replaces the drivers page")
	}
}

func TestTriangleFrameMatchesLibrary(t *testing.T) {
	srv := httptest.NewServer(newPage(t.Context()))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/triangle/frame?turn=0.25&w=24&h=16")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	got, err := png.Decode(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	want := image.NewRGBA(image.Rect(0, 0, 24, 16))
	lewimage.TriangleTurn(want, 0.25)
	if got.Bounds() != want.Bounds() {
		t.Fatalf("bounds %v", got.Bounds())
	}
	for y := 0; y < 16; y++ {
		for x := 0; x < 24; x++ {
			if got.At(x, y) != want.At(x, y) {
				t.Fatalf("pixel %d,%d", x, y)
			}
		}
	}
}
