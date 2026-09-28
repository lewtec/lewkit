// Package version holds build identity for the eletrocromo CLI and packaging.
//
// Goreleaser / go build inject via -ldflags -X (same pattern as most Go CLIs):
//
//	-X github.com/lewtec/lewkit/x/build/version.Version={{.Version}}
//	-X github.com/lewtec/lewkit/x/build/version.Commit={{.Commit}}
//	-X github.com/lewtec/lewkit/x/build/version.Date={{.Date}}
//	-X github.com/lewtec/lewkit/x/build/version.BuiltBy=goreleaser
//
// When those are left at defaults, Resolve fills what it can from
// runtime/debug.BuildInfo (module version + vcs.* when built with VCS stamping)
// and, for a given module directory, from git describe / rev-list.
package version

import (
	"context"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

func getwd() (string, error) { return os.Getwd() }

// Set by -ldflags -X at link time (goreleaser, CI, or make).
var (
	Version = "devel"
	Commit  = ""
	Date    = ""
	BuiltBy = ""
)

// Info is a resolved snapshot.
type Info struct {
	Version string // e.g. v1.2.3, 1.2.3-5-gabcdef, devel
	Commit  string // full or short SHA
	Date    string // RFC3339 or git author date when available
	BuiltBy string // goreleaser, go, git, …
}

// String is a one-line summary suitable for `eletrocromo version`.
func (i Info) String() string {
	parts := []string{i.Version}
	if i.Commit != "" {
		c := i.Commit
		if len(c) > 12 {
			c = c[:12]
		}
		parts = append(parts, "commit="+c)
	}
	if i.Date != "" {
		parts = append(parts, "date="+i.Date)
	}
	if i.BuiltBy != "" {
		parts = append(parts, "builtBy="+i.BuiltBy)
	}
	return strings.Join(parts, " ")
}

// stampedFromLdflags returns Info from -X package vars (Version defaults to "devel").
func stampedFromLdflags() Info {
	info := Info{
		Version: strings.TrimSpace(Version),
		Commit:  strings.TrimSpace(Commit),
		Date:    strings.TrimSpace(Date),
		BuiltBy: strings.TrimSpace(BuiltBy),
	}
	if info.Version == "" {
		info.Version = "devel"
	}
	return info
}

// Resolve returns CLI/binary version from -X vars, then buildinfo VCS,
// then git in the current working directory (local `go run` convenience).
func Resolve() Info {
	info := stampedFromLdflags()
	fillFromBuildInfo(&info)
	if cwd, err := osGetwd(); err == nil {
		fillFromGit(&info, cwd)
	}
	return info
}

// ResolveDir is like Resolve but prefers git metadata from dir (app module root).
// Use this when stamping an APK for a specific project tree.
func ResolveDir(dir string) Info {
	info := stampedFromLdflags()
	fillFromBuildInfo(&info)
	fillFromGit(&info, dir)
	return info
}

// osGetwd is os.Getwd; tests may override.
var osGetwd = func() (string, error) {
	return getwd()
}

func fillFromBuildInfo(info *Info) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if isDevel(info.Version) && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.Version = bi.Main.Version
		if info.BuiltBy == "" {
			info.BuiltBy = "module"
		}
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			if info.Commit == "" {
				info.Commit = s.Value
			}
		case "vcs.time":
			if info.Date == "" {
				info.Date = s.Value
			}
		case "vcs.modified":
			if s.Value == "true" && info.Version != "" && !strings.Contains(info.Version, "dirty") {
				// annotate only when we have a real version string
				if !isDevel(info.Version) {
					info.Version += "-dirty"
				}
			}
		}
	}
}

func fillFromGit(info *Info, dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	if isDevel(info.Version) {
		if desc, err := gitOutput(dir, "describe", "--tags", "--always", "--dirty"); err == nil && desc != "" {
			info.Version = desc
			if info.BuiltBy == "" {
				info.BuiltBy = "git"
			}
		}
	}
	if info.Commit == "" {
		if sha, err := gitOutput(dir, "rev-parse", "HEAD"); err == nil {
			info.Commit = sha
		}
	}
	if info.Date == "" {
		if d, err := gitOutput(dir, "log", "-1", "--format=%cI"); err == nil {
			info.Date = d
		}
	}
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := execdriver.MustCommand("git", args...)
	cmd.Dir = dir
	out, err := execdriver.Output(context.Background(), cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func isDevel(v string) bool {
	v = strings.TrimSpace(v)
	return v == "" || v == "devel" || v == "(devel)"
}

// AndroidName is versionName: strip leading v, replace invalid path-ish bits.
// Empty/devel → "0.0.0-devel".
func (i Info) AndroidName() string {
	v := strings.TrimSpace(i.Version)
	v = strings.TrimPrefix(v, "v")
	if isDevel(v) || v == "" {
		return "0.0.0-devel"
	}
	// Android versionName is free-form but keep it simple for Play.
	v = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.' || r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, v)
	return v
}

// semverCore matches major.minor.patch at the start of a version string.
var semverCore = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)

// AndroidCode is versionCode for Play (positive int, monotonic preference).
//
// Priority:
//  1. semver major/minor/patch → MMmmpp (major*1_000_000 + minor*1_000 + patch)
//  2. git rev-list --count HEAD when dir was used via WithGitCount
//  3. 1
//
// Call AndroidCodeFrom with optional commitCount from git when available.
func (i Info) AndroidCode() int {
	return AndroidCodeFrom(i.Version, 0)
}

// AndroidCodeFrom maps version (+ optional git commit count) to versionCode.
func AndroidCodeFrom(version string, gitCommitCount int) int {
	if m := semverCore.FindStringSubmatch(strings.TrimSpace(version)); len(m) == 4 {
		// Regex already matched digits; Atoi cannot fail for these groups.
		maj, errMaj := strconv.Atoi(m[1])
		min, errMin := strconv.Atoi(m[2])
		pat, errPat := strconv.Atoi(m[3])
		if errMaj != nil || errMin != nil || errPat != nil {
			return 1
		}
		// Cap components so we stay in a reasonable int range.
		if maj > 2099 {
			maj = 2099
		}
		if min > 999 {
			min = 999
		}
		if pat > 999 {
			pat = 999
		}
		code := maj*1_000_000 + min*1_000 + pat
		if code > 0 {
			return code
		}
	}
	if gitCommitCount > 0 {
		return gitCommitCount
	}
	return 1
}

// GitCommitCount returns rev-list --count HEAD in dir, or 0 on error.
func GitCommitCount(dir string) int {
	out, err := gitOutput(dir, "rev-list", "--count", "HEAD")
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

const ldflagsPackage = "github.com/lewtec/lewkit/x/build/version"

// GoBuildLdflags is a single -ldflags value: strip + goreleaser-style -X stamps.
func (i Info) GoBuildLdflags() string {
	var b strings.Builder
	b.WriteString("-s -w")
	writeX := func(name, val string) {
		if val == "" && name != "Version" {
			return
		}
		if val == "" {
			val = "devel"
		}
		b.WriteString(" -X ")
		b.WriteString(ldflagsPackage)
		b.WriteByte('.')
		b.WriteString(name)
		b.WriteByte('=')
		b.WriteString(sanitizeLdflag(val))
	}
	writeX("Version", i.Version)
	writeX("Commit", i.Commit)
	writeX("Date", i.Date)
	writeX("BuiltBy", i.BuiltBy)
	return b.String()
}

// WithAppID appends the x/release stamps for this binary.
// Version is written to x/release.version. appID is written to x/release.appID.
func (i Info) WithAppID(appID string) string {
	flags := i.GoBuildLdflags()
	if versionName := sanitizeLdflag(i.Version); versionName != "" {
		flags += " -X github.com/lewtec/lewkit/x/release.version=" + versionName
	}
	appID = sanitizeLdflag(appID)
	if appID == "" {
		return flags
	}
	return flags + " -X github.com/lewtec/lewkit/x/release.appID=" + appID
}

func sanitizeLdflag(s string) string {
	// -X values must not contain spaces when passed as one argv token.
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' {
			return -1
		}
		return r
	}, s)
}

// NowRFC3339 is a helper for tools that set Date when missing.
func NowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
