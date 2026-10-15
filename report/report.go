package report

import (
	"errors"

	"golang.org/x/sync/errgroup"

	"github.com/lewtec/lewkit/x/errs"
)

// ErrDontReport marks an error that reporters must not send.
// DoNotReport attaches it without changing the error text.
var ErrDontReport = errors.New("do not report")

// DoNotReport returns err with the same text.
// errors.Is(DoNotReport(err), ErrDontReport) is true, and so is the original chain.
// A nil error stays nil. An error that already matches is returned as itself.
func DoNotReport(err error) error {
	if err == nil || errors.Is(err, ErrDontReport) {
		return err
	}
	return errs.Wrap(err, err.Error(), ErrDontReport)
}

var reporters []Reporter

func RegisterReporter(reporter Reporter) {
	reporters = append(reporters, reporter)
}

// Reporter something that reports an error (ex: Sentry)
type Reporter interface {
	Report(err error) error
}

// Report reports an error to all reporters.
// A nil error, and an error that matches ErrDontReport, are ignored.
func Report(err error) {
	if err == nil || errors.Is(err, ErrDontReport) {
		return
	}
	g := new(errgroup.Group)
	for _, reporter := range reporters {
		reporter := reporter
		err := err
		g.Go(func() error {
			return reporter.Report(err)
		})
	}
	err = g.Wait()
	if err != nil {
		panic(err)
	}
}
