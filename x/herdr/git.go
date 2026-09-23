package herdr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var errGit = errors.New("git")

// Info is one git checkout.
type Info struct {
	Toplevel string
	Root     string
	Linked   bool
}

// Worktree is one row from git worktree list.
type Worktree struct {
	Path   string
	Branch string
}

// Git runs git and caches Info and Branch until Clear.
type Git struct {
	mu     sync.Mutex
	info   map[string]infoEntry
	branch map[string]branchEntry
}

type infoEntry struct {
	info Info
	ok   bool
}

type branchEntry struct {
	name string
	ok   bool
}

// Clear drops cached git lookups.
func (g *Git) Clear() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.info = nil
	g.branch = nil
}

func (g *Git) run(ctx context.Context, repo string, args ...string) (string, string, int) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		code = 1
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		}
	}
	return stdout.String(), stderr.String(), code
}

// Info resolves path to its toplevel, common repo root, and linked bit.
func (g *Git) Info(ctx context.Context, path string) (Info, bool) {
	if path == "" {
		return Info{}, false
	}
	key := resolve(path)
	g.mu.Lock()
	if g.info != nil {
		if hit, ok := g.info[key]; ok {
			g.mu.Unlock()
			return hit.info, hit.ok
		}
	}
	g.mu.Unlock()
	stdout, _, code := g.run(ctx, key, "rev-parse", "--path-format=absolute", "--show-toplevel", "--git-common-dir")
	var hit infoEntry
	if code == 0 {
		lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
		if len(lines) >= 2 {
			toplevel := resolve(lines[0])
			common := resolve(lines[1])
			root := common
			if filepath.Base(common) == ".git" {
				root = filepath.Dir(common)
			}
			hit = infoEntry{info: Info{Toplevel: toplevel, Root: root, Linked: toplevel != root}, ok: true}
		}
	}
	g.mu.Lock()
	if g.info == nil {
		g.info = map[string]infoEntry{}
	}
	g.info[key] = hit
	g.mu.Unlock()
	return hit.info, hit.ok
}

// Branch is the checked-out branch, or ok false when detached or missing.
func (g *Git) Branch(ctx context.Context, path string) (string, bool) {
	if path == "" {
		return "", false
	}
	key := resolve(path)
	g.mu.Lock()
	if g.branch != nil {
		if hit, ok := g.branch[key]; ok {
			g.mu.Unlock()
			return hit.name, hit.ok
		}
	}
	g.mu.Unlock()
	stdout, _, code := g.run(ctx, key, "rev-parse", "--abbrev-ref", "HEAD")
	name := strings.TrimSpace(stdout)
	hit := branchEntry{name: name, ok: code == 0 && name != "" && name != "HEAD"}
	g.mu.Lock()
	if g.branch == nil {
		g.branch = map[string]branchEntry{}
	}
	g.branch[key] = hit
	g.mu.Unlock()
	return hit.name, hit.ok
}

// Dirty reports uncommitted changes, including untracked files.
func (g *Git) Dirty(ctx context.Context, repo string) bool {
	stdout, _, code := g.run(ctx, repo, "status", "--porcelain")
	return code == 0 && strings.TrimSpace(stdout) != ""
}

// DefaultBranch is origin/HEAD when that is main or master, else a local main or master.
func (g *Git) DefaultBranch(ctx context.Context, repo string) (string, bool) {
	stdout, _, code := g.run(ctx, repo, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD")
	if code == 0 {
		name := stdout
		if i := strings.LastIndex(strings.TrimSpace(name), "/"); i >= 0 {
			name = name[i+1:]
		}
		name = strings.TrimSpace(name)
		if name == "main" || name == "master" {
			return name, true
		}
	}
	for _, name := range []string{"master", "main"} {
		_, _, code := g.run(ctx, repo, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
		if code == 0 {
			return name, true
		}
	}
	return "", false
}

// Worktrees parses git worktree list --porcelain.
func (g *Git) Worktrees(ctx context.Context, repo string) []Worktree {
	stdout, _, code := g.run(ctx, repo, "worktree", "list", "--porcelain")
	if code != 0 {
		return nil
	}
	return parseWorktrees(stdout)
}

func parseWorktrees(stdout string) []Worktree {
	var rows []Worktree
	var path, branch string
	has := false
	flush := func() {
		if has {
			rows = append(rows, Worktree{Path: resolve(path), Branch: branch})
		}
		path, branch, has = "", "", false
	}
	for line := range strings.SplitSeq(stdout, "\n") {
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "worktree "):
			flush()
			path = line[len("worktree "):]
			has = true
		case strings.HasPrefix(line, "branch "):
			ref := line[len("branch "):]
			const prefix = "refs/heads/"
			branch = ref
			if strings.HasPrefix(ref, prefix) {
				branch = ref[len(prefix):]
			}
		case line == "detached":
			branch = ""
		}
	}
	flush()
	return rows
}

// HasRef reports whether ref exists.
func (g *Git) HasRef(ctx context.Context, repo, ref string) bool {
	_, _, code := g.run(ctx, repo, "show-ref", "--verify", "--quiet", ref)
	return code == 0
}

// Run runs git in repo and returns stderr on failure.
func (g *Git) Run(ctx context.Context, repo string, args ...string) error {
	_, stderr, code := g.run(ctx, repo, args...)
	if code != 0 {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = fmt.Sprintf("exit %d", code)
		}
		return fmt.Errorf("%w -C %s %s: %s", errGit, repo, strings.Join(args, " "), msg)
	}
	return nil
}

// OriginSlug is owner-name from origin, lowercased.
func (g *Git) OriginSlug(ctx context.Context, repo string) string {
	stdout, _, code := g.run(ctx, repo, "remote", "get-url", "origin")
	if code != 0 {
		return ""
	}
	return originSlug(strings.TrimSpace(stdout))
}

func originSlug(url string) string {
	url = strings.TrimSuffix(url, ".git")
	var owner, name string
	if i := strings.Index(url, "://"); i >= 0 {
		parts := strings.Split(url[i+3:], "/")
		if len(parts) < 3 {
			return ""
		}
		owner, name = parts[len(parts)-2], parts[len(parts)-1]
	} else if _, rest, ok := strings.Cut(url, ":"); ok {
		rest = strings.TrimPrefix(rest, "/")
		owner, name, ok = strings.Cut(rest, "/")
		if !ok {
			return ""
		}
	} else {
		return ""
	}
	if owner == "" || name == "" {
		return ""
	}
	return strings.ToLower(owner + "-" + name)
}

// GrokSlug is the worktree directory name under grokRoot, or origin, or the repo name.
func (g *Git) GrokSlug(ctx context.Context, repo, grokRoot string) string {
	counts := map[string]int{}
	root := resolve(grokRoot)
	for _, wt := range g.Worktrees(ctx, repo) {
		rel, err := filepath.Rel(root, wt.Path)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}
		slug := strings.SplitN(rel, string(filepath.Separator), 2)[0]
		if slug != "" {
			counts[slug]++
		}
	}
	best, n := "", 0
	for slug, count := range counts {
		if count > n || (count == n && slug > best) {
			best, n = slug, count
		}
	}
	if best != "" {
		return best
	}
	if slug := g.OriginSlug(ctx, repo); slug != "" {
		return slug
	}
	return strings.ToLower(filepath.Base(repo))
}

// LinkedWorktree is an existing linked checkout of branch, other than repo itself.
func (g *Git) LinkedWorktree(ctx context.Context, repo, branch string) (string, bool) {
	root := resolve(repo)
	for _, wt := range g.Worktrees(ctx, repo) {
		if wt.Branch == branch && wt.Path != root {
			return wt.Path, true
		}
	}
	return "", false
}

func resolve(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real
	}
	return abs
}
