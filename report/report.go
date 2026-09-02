package report

import "golang.org/x/sync/errgroup"

var reporters []Reporter

func RegisterReporter(reporter Reporter) {
	reporters = append(reporters, reporter)
}

// Reporter something that reports an error (ex: Sentry)
type Reporter interface {
	Report(err error) error
}

// Report reports an error to all reporters
func Report(err error) {
	if err == nil {
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
