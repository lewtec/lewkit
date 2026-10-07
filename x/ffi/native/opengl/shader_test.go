//go:build !darwin && !ios

package opengl

import (
	"strings"
	"testing"
)

func TestVersionHeader(t *testing.T) {
	if got := versionHeader(&glAPI{major: 3, minor: 3}); !strings.Contains(got, "#version 330 core") {
		t.Fatalf("3.3 header %q", got)
	}
	if got := versionHeader(&glAPI{major: 4, minor: 6}); !strings.Contains(got, "#version 330 core") {
		t.Fatalf("4.6 header %q", got)
	}
	if got := versionHeader(&glAPI{major: 4, minor: 1}); !strings.Contains(got, "#version 410") {
		t.Fatalf("4.1 header %q", got)
	}
	if got := versionHeader(&glAPI{gles: true, major: 3, minor: 0}); !strings.Contains(got, "#version 300 es") {
		t.Fatalf("es header %q", got)
	}
}

func TestShadersFlipY(t *testing.T) {
	for _, g := range []*glAPI{
		{gles: true, major: 3, minor: 0},
		{major: 3, minor: 3},
	} {
		fill := fillFragment(g)
		if strings.Contains(fill, "origin_upper_left") {
			t.Fatal("fill uses origin_upper_left")
		}
		if !strings.Contains(fill, "uExtent.y - gl_FragCoord.y") {
			t.Fatal("fill does not flip y")
		}
		image := imageFragment(g)
		if strings.Contains(image, "origin_upper_left") {
			t.Fatal("image uses origin_upper_left")
		}
		if !strings.Contains(image, "size.y - 1 - int(gl_FragCoord.y)") {
			t.Fatal("image does not flip y")
		}
	}
}
