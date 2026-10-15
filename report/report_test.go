package report

import (
	"errors"
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
