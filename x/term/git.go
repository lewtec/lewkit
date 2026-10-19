package term

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/git"
)

type gitMsg struct {
	dir  string
	line string
}

// readGit is the one-line git status of dir.
// A missing repository is a quiet sentence. The branch and the changed
// count come from git status --porcelain -b.
func readGit(ctx context.Context, dir string) string {
	if dir == "" {
		return "no history here"
	}
	if err := ctx.Err(); err != nil {
		return "no history here"
	}
	out, err := (&git.Git{}).Output(ctx, dir, "status", "--porcelain", "-b")
	if err != nil {
		return "no history here"
	}
	return gitSummary(out)
}

// gitSummary reads the porcelain -b text.
// The first line is the branch. Later lines are changes.
func gitSummary(status string) string {
	status = strings.TrimRight(status, "\n")
	if status == "" {
		return "no history here"
	}
	lines := strings.Split(status, "\n")
	if !strings.HasPrefix(lines[0], "## ") {
		return "no history here"
	}
	branch := strings.TrimPrefix(lines[0], "## ")
	if strings.HasPrefix(branch, "No commits yet on ") {
		branch = strings.TrimPrefix(branch, "No commits yet on ")
	}
	if i := strings.Index(branch, "..."); i >= 0 {
		branch = branch[:i]
	}
	branch = strings.TrimSpace(branch)
	if branch == "" || branch == "HEAD" || strings.HasPrefix(branch, "HEAD ") {
		branch = "detached"
	}
	changed := 0
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) != "" {
			changed++
		}
	}
	if changed == 0 {
		return branch + " · tidy"
	}
	if changed == 1 {
		return branch + " · 1 changed"
	}
	return branch + " · " + strconv.Itoa(changed) + " changed"
}

func roomLabel(dir string) string {
	if dir == "" {
		return "nowhere yet"
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return dir
	}
	if dir == home {
		return "~"
	}
	if strings.HasPrefix(dir, home+"/") {
		return "~" + strings.TrimPrefix(dir, home)
	}
	return dir
}
