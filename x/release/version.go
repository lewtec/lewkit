package release

import (
	"cmp"
	"runtime"
	"runtime/debug"
	"strings"
)

var version = "dev"

func formatVersion(version, vcs string) string {
	parts := []string{
		cmp.Or(version, "dev"),
	}
	if vcs != "" {
		parts = append(parts, vcs)
	}
	return strings.Join(parts, "-")
}

func Version() string {
	version := strings.TrimSpace(version)
	vcs := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				vcs = setting.Value[:8]
			}
		}
	}
	return formatVersion(version, vcs)
}

// Platform gives GOOS-GOARCH-MICROARCH
func Platform() string {
	p := runtime.GOOS + "-" + runtime.GOARCH
	if m := microarch(); m != "" {
		return p + "-" + m
	}
	return p
}

// microarch returns the GO$GOARCH microarchitecture level recorded at build time.
func microarch() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "GOAMD64", "GOARM", "GOARM64", "GO386",
			"GOMIPS", "GOMIPS64", "GOPPC64", "GORISCV64", "GOWASM":
			return s.Value
		}
	}
	return ""
}
