package gotool

import (
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/toolchain/tamago"
	"github.com/stretchr/testify/require"
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
	require.ErrorIs(t, err, ErrTarget)
	body, err := target.ImportCfg([]Object{{
		ImportPath: "main",
		Archive:    "/main.a",
	}})
	require.NoError(t, err)
	if !strings.Contains(body, "packagefile main=/main.a\n") || !strings.Contains(body, "packagefile runtime=/rt.a\n") {
		require.FailNowf(t, "", "importcfg:\n%s", body)
	}
	image, err := target.LinkImage("/out", "/cfg", Object{ImportPath: "main", Archive: "/main.a"})
	require.NoError(t, err)
	if image.Mode != "exe" || image.Archive != "/main.a" || image.Out != "/out" {
		require.FailNowf(t, "", "image %+v", image)
	}
	if strings.Join(image.Ldflags, " ") != strings.Join(tamago.LinkFlags, " ") {
		require.FailNowf(t, "", "ldflags %v", image.Ldflags)
	}
	_, err = (Target{}).ImportCfg(nil)
	require.ErrorIs(t, err, ErrEmptyTarget)
	err = BuildObjects(t.Context(), []Unit{{
		ImportPath: "main",
		Files:      []string{"m.go"},
		Archive:    "m.a",
		ImportCfg:  "m.cfg",
		GOOS:       "linux",
		GOARCH:     "amd64",
	}}, target, "/out")
	require.ErrorIs(t, err, ErrTarget)
}

func TestPlanRejectsAssembly(t *testing.T) {
	_, err := Plan([]Unit{{
		ImportPath: "p",
		Files:      []string{"p.go"},
		Archive:    "p.a",
		ImportCfg:  "p.cfg",
		Asm:        []string{"p.s"},
	}}, nil)
	require.ErrorIs(t, err, ErrNoAsm)
}
