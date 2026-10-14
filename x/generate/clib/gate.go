package clib

import (
	"fmt"
	"go/build/constraint"
	"os"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/generate/header"
)

// hosts is the generate and link gate. Each row is one GOOS/GOARCH this
// package writes. link is the internal PIE shared library. A pair that is
// absent is ErrHost at generate time. A row with link false still generates,
// and BuildFor returns ErrHost before the linker.
func hosts() []target {
	return []target{
		{goos: "linux", goarch: "amd64", mach: machAmd64, constraint: "(linux || android) && amd64", link: true},
		{goos: "linux", goarch: "arm64", mach: machArm64, constraint: "(linux || android) && arm64", link: true},
		{goos: "linux", goarch: "386", mach: mach386, constraint: "(linux || android) && 386"},
		{goos: "linux", goarch: "arm", mach: machARM, constraint: "(linux || android) && arm"},
		{goos: "android", goarch: "amd64", mach: machAmd64, constraint: "(linux || android) && amd64"},
		{goos: "android", goarch: "arm64", mach: machArm64, constraint: "(linux || android) && arm64", link: true},
		{goos: "android", goarch: "386", mach: mach386, constraint: "(linux || android) && 386"},
		{goos: "android", goarch: "arm", mach: machARM, constraint: "(linux || android) && arm"},
		{goos: "windows", goarch: "amd64", mach: machWin64, constraint: "windows && amd64", fileOS: "windows", link: true},
		{goos: "windows", goarch: "arm64", mach: machArm64, constraint: "windows && arm64", fileOS: "windows", link: true},
		{goos: "windows", goarch: "386", mach: machWin386, constraint: "windows && 386", fileOS: "windows", link: true},
		{goos: "darwin", goarch: "amd64", mach: machAmd64, constraint: "darwin && amd64", fileOS: "darwin", link: true},
		{goos: "darwin", goarch: "arm64", mach: machArm64, constraint: "darwin && arm64", fileOS: "darwin", link: true},
	}
}

func hostFor(goos, goarch string) (target, error) {
	for _, h := range hosts() {
		if h.goos == goos && h.goarch == goarch {
			return h, nil
		}
	}
	return target{}, fmt.Errorf("%w: %s/%s", ErrHost, goos, goarch)
}

func hostsByOS(goos string) []target {
	var out []target
	for _, h := range hosts() {
		if h.goos == goos {
			out = append(out, h)
		}
	}
	return out
}

func targetsFor(spec string) ([]target, error) {
	switch spec {
	case "", "all":
		return hostsByOS("linux"), nil
	case "windows":
		return hostsByOS("windows"), nil
	case "darwin":
		return hostsByOS("darwin"), nil
	case "host":
		spec = runtime.GOOS + "/" + runtime.GOARCH
	case "sysv64":
		spec = "linux/amd64"
	case "386":
		spec = "linux/386"
	case "arm":
		spec = "linux/arm"
	case "win64":
		spec = "windows/amd64"
	case "win386":
		spec = "windows/386"
	case "darwinarm64":
		spec = "darwin/arm64"
	}
	parsed, err := header.ParseABI(spec)
	if err != nil {
		return nil, err
	}
	if parsed.GOOS == "" || parsed.GOARCH == "" {
		return nil, fmt.Errorf("%w: %s", ErrHost, spec)
	}
	t, err := hostFor(parsed.GOOS, parsed.GOARCH)
	if err != nil {
		return nil, err
	}
	return []target{t}, nil
}

// exportCovers reports whether the //go:build line of the generated export
// file includes goos/goarch. A tag is true only when it is that GOOS or
// GOARCH. android does not imply linux here: the emitter names both.
func exportCovers(path, goos, goarch string) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var line string
	for _, item := range strings.Split(string(buf), "\n") {
		if strings.HasPrefix(item, "//go:build ") {
			line = item
			break
		}
	}
	miss := fmt.Errorf("%w: generated export does not include %s/%s", ErrHost, goos, goarch)
	if line == "" {
		return miss
	}
	expr, err := constraint.Parse(line)
	if err != nil {
		return miss
	}
	if !expr.Eval(func(tag string) bool {
		return tag == goos || tag == goarch
	}) {
		return miss
	}
	return nil
}

// requireImageExports fails when a cgo_export_dynamic name is absent from
// the linked image. The error names the symbol.
func requireImageExports(path, goos string, names []string) error {
	switch goos {
	case "windows":
		buf, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		have, err := peExports(buf)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", ErrRequireImageExportsClib, path, err)
		}
		return missingExport(path, names, have)
	case "darwin":
		trie, err := machoExportTrie(path)
		if err != nil {
			return err
		}
		for _, name := range names {
			addr, ok := trieLookup(trie, name)
			if !ok || addr == 0 {
				return fmt.Errorf("clib: %s: export %s %w", path, name, ErrIsNotInTheImage)
			}
		}
		return nil
	default:
		have, err := elfExports(path)
		if err != nil {
			return fmt.Errorf("%w: %s: %w", ErrRequireImageExportsClib, path, err)
		}
		return missingExport(path, names, have)
	}
}

func missingExport(path string, names []string, have map[string]bool) error {
	for _, name := range names {
		if !have[name] {
			return fmt.Errorf("clib: %s: export %s %w", path, name, ErrIsNotInTheImage)
		}
	}
	return nil
}
