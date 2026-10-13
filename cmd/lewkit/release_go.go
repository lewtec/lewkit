package main

import (
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
)

// goCmd is `lewkit release go`. Flags may sit before or after the
// package arguments. `--` ends flags.
type goCmd struct {
	build *goBuildCmd
	run   *goRunCmd
}

func (goCmd) Description() string { return "compile with the embedded Go toolchain" }

type goBuildCmd struct {
	dir     cmd.StringArg   `long:"C" short:"C" default:"" help:"directory to build in"`
	output  cmd.StringArg   `long:"o" short:"o" default:"" help:"output file"`
	tags    cmd.StringArg   `long:"tags" default:"" help:"comma-separated build tags"`
	ldflags cmd.StringArg   `long:"ldflags" default:"" help:"extra linker arguments"`
	pkgdir  cmd.StringArg   `long:"pkgdir" default:"" help:"directory of standard-library archives"`
	pkgs    []cmd.StringArg `help:"packages or .go files"`
}

func (goBuildCmd) Description() string { return "compile packages and link a main package" }

type goRunCmd struct {
	goBuildCmd `flatten:""`
	sep        *cmd.Dash
	args       []cmd.StringArg `help:"arguments for the program"`
}

func (goRunCmd) Description() string { return "compile, link, and run a main package" }

// normalizeReleaseGoFlags rewrites Go's single-dash long flags after
// `release go`. -tags, -ldflags, and -pkgdir become the long form the
// parser reads. Tokens after `--` stay as the program typed them.
func normalizeReleaseGoFlags(args []string) []string {
	seenRelease := false
	seenGo := false
	out := make([]string, len(args))
	for i, a := range args {
		if a == "--" {
			copy(out[i:], args[i:])
			return out
		}
		if !seenGo && !strings.HasPrefix(a, "-") {
			if a == "release" {
				seenRelease = true
			} else if seenRelease && a == "go" {
				seenGo = true
			}
		}
		if seenGo {
			a = rewriteGoFlag(a)
		}
		out[i] = a
	}
	return out
}

func rewriteGoFlag(a string) string {
	if !strings.HasPrefix(a, "-") || strings.HasPrefix(a, "--") || len(a) < 3 {
		return a
	}
	name, val, has := strings.Cut(a[1:], "=")
	switch name {
	case "tags", "ldflags", "pkgdir":
	default:
		return a
	}
	if has {
		return "--" + name + "=" + val
	}
	return "--" + name
}
