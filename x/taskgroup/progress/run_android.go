//go:build android

package progress

import (
	"context"

	"github.com/lewtec/lewkit/x/taskgroup"
)

func runTea(s *taskgroup.Session, ctx context.Context, work func(context.Context) error) error {
	err := work(ctx)
	if werr := s.Wait(); err == nil {
		err = werr
	}
	return err
}
