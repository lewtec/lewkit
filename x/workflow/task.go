package workflow

import (
	"context"
	"fmt"
	"io"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/tool"
)

type dirKey struct{}

func withDir(ctx context.Context, dir string) context.Context {
	if dir == "" {
		return ctx
	}
	return context.WithValue(ctx, dirKey{}, dir)
}

// Dir is the directory of the step currently running.
// A task uses it for paths that belong next to that step's inputs.
func Dir(ctx context.Context) string {
	dir, _ := ctx.Value(dirKey{}).(string)
	return dir
}

// Task is one unit of work inside a step.
// A shell command, a download, an extract, and an in-process function
// are the kinds. Run is called only after the step is out of date.
type Task interface {
	Run(ctx context.Context, st *taskgroup.Status) error
}

// Command runs one shell line through the exec driver.
// An empty Dir uses the step directory. A non-empty Env replaces the
// process environment. Ignore keeps going when the line fails.
type Command struct {
	Text   string
	Dir    string
	Env    []string
	Ignore bool
}

// Run starts the shell line. An empty line does nothing.
// Stdout is a progress line writer. Stderr stays unset so the exec
// driver attaches the taskgroup hook.
func (c Command) Run(ctx context.Context, _ *taskgroup.Status) error {
	if strings.TrimSpace(c.Text) == "" {
		return nil
	}
	cmd := execdriver.MustCommand(ctx, "sh", "-c", c.Text)
	dir := c.Dir
	if dir == "" {
		dir = Dir(ctx)
	}
	if dir != "" {
		cmd.Dir = dir
	}
	if len(c.Env) > 0 {
		cmd.Env = c.Env
	}
	stdout := taskgroup.LineWriterFrom(ctx)
	cmd.Stdout = stdout
	defer closeWriter(stdout)
	err := execdriver.Run(ctx, cmd)
	if err != nil && !c.Ignore {
		return err
	}
	return nil
}

// String is the shell line, used as the step title when Desc is empty.
func (c Command) String() string { return c.Text }

// Func is an in-process task. The function has the same shape as the
// callback passed to taskgroup.Go.
type Func func(context.Context, *taskgroup.Status) error

// Run calls the function. A nil function does nothing.
func (f Func) Run(ctx context.Context, st *taskgroup.Status) error {
	if f == nil {
		return nil
	}
	return f(ctx, st)
}

// Download fetches URL into Dest with tool.DownloadFile.
// A relative Dest is joined with the step directory.
// Options.Hash empty uses the HTTP client. A hash uses fetchurl.
type Download struct {
	URL     string
	Dest    string
	Options tool.DownloadOptions
}

// Run writes the file. An empty destination is refused.
func (d Download) Run(ctx context.Context, _ *taskgroup.Status) error {
	if strings.TrimSpace(d.Dest) == "" {
		return fmt.Errorf("download: empty destination")
	}
	return tool.DownloadFile(ctx, d.URL, Join(Dir(ctx), d.Dest), d.Options)
}

// String is the download, used as the step title when Desc is empty.
func (d Download) String() string {
	if d.URL == "" {
		return "download"
	}
	return "download " + d.URL
}

// Extract unpacks Source into Dest with tool.Extract.
// Relative paths are joined with the step directory.
// Zip, squashfs, and tar are unpacked. Anything else is copied as one file.
type Extract struct {
	Source string
	Dest   string
}

// Run unpacks the archive.
func (e Extract) Run(ctx context.Context, _ *taskgroup.Status) error {
	return tool.Extract(ctx, Join(Dir(ctx), e.Source), Join(Dir(ctx), e.Dest))
}

// String is the extract, used as the step title when Desc is empty.
func (e Extract) String() string {
	if e.Source == "" {
		return "extract"
	}
	return "extract " + e.Source
}

func closeWriter(w io.WriteCloser) {
	if w != nil {
		_ = w.Close()
	}
}
