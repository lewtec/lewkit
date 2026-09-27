package taskgroup

import (
	"context"
	"io"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

func init() {
	execdriver.SetStderr(func(ctx context.Context) io.Writer {
		return LineWriterFrom(ctx)
	})
}
