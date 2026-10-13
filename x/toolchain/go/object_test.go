package gotool

import (
	"errors"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/toolchain/tamago"
)

func TestTargetImportCfg(t *testing.T) {
	target := Target{
		GOOS:    "tamago",
		GOARCH:  "amd64",
		Mode:    "exe",
		Ldflags: append([]string(nil), tamago.LinkFlags...),
		Runtime: []Object{{
			ImportPath: "runtime",
			Archive:    "/rt.a",
			GOOS:       "tamago",
			GOARCH:     "amd64",
		}},
	}
	_, err := target.ImportCfg([]Object{{
		ImportPath: "main",
		Archive:    "/main.a",
		GOOS:       "linux",
		GOARCH:     "amd64",
	}})
	if !errors.Is(err, ErrTarget) {
		t.Fatalf("mixed target: %v", err)
	}
	body, err := target.ImportCfg([]Object{{
		ImportPath: "main",
		Archive:    "/main.a",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "packagefile main=/main.a\n") || !strings.Contains(body, "packagefile runtime=/rt.a\n") {
		t.Fatalf("importcfg:\n%s", body)
	}
	image, err := target.LinkImage("/out", "/cfg", Object{ImportPath: "main", Archive: "/main.a"})
	if err != nil {
		t.Fatal(err)
	}
	if image.Mode != "exe" || image.Archive != "/main.a" || image.Out != "/out" {
		t.Fatalf("image %+v", image)
	}
	if strings.Join(image.Ldflags, " ") != strings.Join(tamago.LinkFlags, " ") {
		t.Fatalf("ldflags %v", image.Ldflags)
	}
	if _, err := (Target{}).ImportCfg(nil); !errors.Is(err, ErrEmptyTarget) {
		t.Fatalf("empty target: %v", err)
	}
	err = BuildObjects(t.Context(), []Unit{{
		ImportPath: "main",
		Files:      []string{"m.go"},
		Archive:    "m.a",
		ImportCfg:  "m.cfg",
		GOOS:       "linux",
		GOARCH:     "amd64",
	}}, target, "/out")
	if !errors.Is(err, ErrTarget) {
		t.Fatalf("build objects: %v", err)
	}
}

func TestPlanRejectsAssembly(t *testing.T) {
	_, err := Plan([]Unit{{
		ImportPath: "p",
		Files:      []string{"p.go"},
		Archive:    "p.a",
		ImportCfg:  "p.cfg",
		Asm:        []string{"p.s"},
	}}, nil)
	if !errors.Is(err, ErrNoAsm) {
		t.Fatalf("asm: %v", err)
	}
}
