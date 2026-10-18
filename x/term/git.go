package term

import (
	"context"
	"io"
	"os"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
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
	cmd := execdriver.MustCommand(ctx, "git", "-C", dir, "status", "--porcelain", "-b")
	cmd.Dir = dir
	cmd.Stderr = io.Discard
	out, err := execdriver.Output(ctx, cmd)
	if err != nil {
		return "no history here"
	}
	return gitSummary(string(out))
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
	return branch + " · " + itoa(changed) + " changed"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
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
