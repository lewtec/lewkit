package common

import (
	"context"
	"fmt"
	"io"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/taskgroup"
)

const xcodeDiagnosticLines = 40

// RunXcodebuild runs xcodebuild in dir.
// A failure wraps ErrXcodebuild with the process error and the diagnostic
// tail: the lines that contain "error:", or the last lines of the log when
// xcodebuild printed none. Stdout and stderr also go to the session line writer.
func RunXcodebuild(ctx context.Context, dir string, args ...string) error {
	if taskgroup.FromContext(ctx) == nil {
		return runXcodebuild(ctx, dir, args...)
	}
	errCh := make(chan error, 1)
	taskgroup.Go(ctx, "xcodebuild", taskgroup.IO, func(ctx context.Context, st *taskgroup.Status) error {
		st.Update("xcodebuild")
		err := runXcodebuild(ctx, dir, args...)
		errCh <- err
		return err
	})
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func runXcodebuild(ctx context.Context, dir string, args ...string) error {
	cmd := execdriver.MustCommand(ctx, "xcodebuild", args...)
	cmd.Dir = dir
	log := &xcodeLog{}
	lines := taskgroup.LineWriterFrom(ctx)
	defer lines.Close()
	cmd.Stdout = io.MultiWriter(lines, log)
	cmd.Stderr = cmd.Stdout
	err := execdriver.Run(ctx, cmd)
	if err == nil {
		return nil
	}
	diag := log.diagnostic()
	if diag == "" {
		return fmt.Errorf("%w: %w", ErrXcodebuild, err)
	}
	return fmt.Errorf("%w: %w\n%s", ErrXcodebuild, err, diag)
}

// xcodeLog keeps the error lines and the trailing lines of an xcodebuild log.
type xcodeLog struct {
	errors []string
	tail   []string
	cur    []byte
}

func (l *xcodeLog) Write(p []byte) (int, error) {
	for _, b := range p {
		if b == '\n' {
			l.commit()
			continue
		}
		if b == '\r' {
			continue
		}
		if len(l.cur) < 4096 {
			l.cur = append(l.cur, b)
		}
	}
	return len(p), nil
}

func (l *xcodeLog) commit() {
	line := strings.TrimSpace(string(l.cur))
	l.cur = l.cur[:0]
	if line == "" {
		return
	}
	l.tail = appendBound(l.tail, line, xcodeDiagnosticLines)
	if strings.Contains(line, "error:") {
		l.errors = appendBound(l.errors, line, xcodeDiagnosticLines)
	}
}

func (l *xcodeLog) diagnostic() string {
	if len(l.cur) > 0 {
		l.commit()
	}
	lines := l.errors
	if len(lines) == 0 {
		lines = l.tail
	}
	return strings.Join(lines, "\n")
}

func appendBound(lines []string, line string, max int) []string {
	lines = append(lines, line)
	if len(lines) <= max {
		return lines
	}
	out := make([]string, max)
	copy(out, lines[len(lines)-max:])
	return out
}
