package report

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

type captureReporter struct {
	calls int
	err   error
}

func (c *captureReporter) Report(err error) error {
	c.calls++
	c.err = err
	return nil
}

func TestDoNotReportKeepsTheText(t *testing.T) {
	err := DoNotReport(errs.Wrap(errs.ErrMissing, "missing --project"))
	require.EqualError(t, err, "missing --project")
	require.ErrorIs(t, err, ErrDontReport)
	require.ErrorIs(t, err, errs.ErrMissing)
	require.NotContains(t, err.Error(), ErrDontReport.Error())

	wrapped := fmt.Errorf("decomp view: %w", err)
	require.EqualError(t, wrapped, "decomp view: missing --project")
	require.ErrorIs(t, wrapped, ErrDontReport)
	require.ErrorIs(t, wrapped, errs.ErrMissing)

	cause := DoNotReport(fmt.Errorf("read: %w", errBoom))
	require.EqualError(t, cause, "read: boom")
	require.ErrorIs(t, cause, errBoom)
	require.ErrorIs(t, cause, ErrDontReport)

	require.NoError(t, DoNotReport(nil))
	require.Equal(t, err, DoNotReport(err))
}

func TestReportSkipsErrDontReport(t *testing.T) {
	previous := append([]Reporter(nil), reporters...)
	t.Cleanup(func() { reporters = previous })
	reporters = nil

	capture := &captureReporter{}
	RegisterReporter(capture)
	Report(DoNotReport(errs.Wrap(errs.ErrMissing, "missing --project")))
	require.Zero(t, capture.calls)

	Report(errBoom)
	require.Equal(t, 1, capture.calls)
	require.ErrorIs(t, capture.err, errBoom)
}

func TestReportSkipsContextCanceled(t *testing.T) {
	previous := append([]Reporter(nil), reporters...)
	t.Cleanup(func() { reporters = previous })
	reporters = nil

	capture := &captureReporter{}
	RegisterReporter(capture)
	Report(context.Canceled)
	Report(fmt.Errorf("wait: %w", context.Canceled))
	require.Zero(t, capture.calls)
}
