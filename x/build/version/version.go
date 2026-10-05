// Package version resolves packaging identity for a module tree.
//
// The runtime binary stamp is x/release.version. This package does not
// export a Version variable. GoBuildLdflags writes that stamp. Commit,
// Date, and BuiltBy are link metadata for Resolve, not a second version.
//
// When those metadata fields are empty, Resolve fills what it can from
// runtime/debug.BuildInfo and, for a module directory, from git describe,
// rev-parse, log, and rev-list through x/git.
package version

import (
	"context"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/git"
)

func getwd() (string, error) { return os.Getwd() }

// Commit, Date, and BuiltBy are set by -ldflags -X. They are build
// metadata. The runtime version stamp is x/release.version.
var (
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

// stampedFromLdflags returns Info from -X metadata. Version starts as
// "devel" until build info or git fills it. There is no Version variable.
func stampedFromLdflags() Info {
	return Info{
		Version: "devel",
		Commit:  strings.TrimSpace(Commit),
		Date:    strings.TrimSpace(Date),
		BuiltBy: strings.TrimSpace(BuiltBy),
	}
}

// Resolve returns CLI/binary version from -X vars, then buildinfo VCS,
// then git in the current working directory (local `go run` convenience).
// ctx is the caller's context. Git reads stop when it ends.
func Resolve(ctx context.Context) Info {
	info := stampedFromLdflags()
	fillFromBuildInfo(&info)
	if cwd, err := osGetwd(); err == nil {
		fillFromGit(ctx, &info, cwd)
	}
	return info
}

// ResolveDir is like Resolve but prefers git metadata from dir (app module root).
// Use this when stamping a package for a specific project tree.
// ctx is the caller's context. Git reads stop when it ends.
func ResolveDir(ctx context.Context, dir string) Info {
	info := stampedFromLdflags()
	fillFromBuildInfo(&info)
	fillFromGit(ctx, &info, dir)
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

func fillFromGit(ctx context.Context, info *Info, dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	if isDevel(info.Version) {
		if desc, err := gitOutput(ctx, dir, "describe", "--tags", "--always", "--dirty"); err == nil && desc != "" {
			info.Version = desc
			if info.BuiltBy == "" {
				info.BuiltBy = "git"
			}
		}
	}
	if info.Commit == "" {
		if sha, err := gitOutput(ctx, dir, "rev-parse", "HEAD"); err == nil {
			info.Commit = sha
		}
	}
	if info.Date == "" {
		if d, err := gitOutput(ctx, dir, "log", "-1", "--format=%cI"); err == nil {
			info.Date = d
		}
	}
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	var g git.Git
	return g.Output(ctx, dir, args...)
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
// ctx is the caller's context.
func GitCommitCount(ctx context.Context, dir string) int {
	out, err := gitOutput(ctx, dir, "rev-list", "--count", "HEAD")
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

const (
	metaPackage    = "github.com/lewtec/lewkit/x/build/version"
	releasePackage = "github.com/lewtec/lewkit/x/release"
)

// GoBuildLdflags is one -ldflags value. The version stamp is
// x/release.version. Commit, Date, and BuiltBy stay metadata on this package.
func (i Info) GoBuildLdflags() string {
	var b strings.Builder
	b.WriteString("-s -w")
	if versionName := sanitizeLdflag(i.Version); versionName != "" {
		b.WriteString(" -X ")
		b.WriteString(releasePackage)
		b.WriteString(".version=")
		b.WriteString(versionName)
	}
	writeMeta := func(name, val string) {
		val = sanitizeLdflag(val)
		if val == "" {
			return
		}
		b.WriteString(" -X ")
		b.WriteString(metaPackage)
		b.WriteByte('.')
		b.WriteString(name)
		b.WriteByte('=')
		b.WriteString(val)
	}
	writeMeta("Commit", i.Commit)
	writeMeta("Date", i.Date)
	writeMeta("BuiltBy", i.BuiltBy)
	return b.String()
}

// WithAppID appends x/release.appID. The version stamp is already
// x/release.version from GoBuildLdflags.
func (i Info) WithAppID(appID string) string {
	flags := i.GoBuildLdflags()
	appID = sanitizeLdflag(appID)
	if appID == "" {
		return flags
	}
	return flags + " -X " + releasePackage + ".appID=" + appID
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
