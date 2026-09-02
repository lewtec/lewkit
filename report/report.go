package report

import "golang.org/x/sync/errgroup"

var reporters map[string]Reporter

func RegisterReporter(reporter Reporter)

// Reporter something that reports an error (ex: Sentry)
type Reporter interface {
	Report(err error) error
	Name() string
}

// Report reports an error to all reporters
func Report(err error) {
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
