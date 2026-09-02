package report

var reporters map[string]Reporter

func RegisterReporter(reporter Reporter)

// Reporter something that reports an error (ex: Sentry)
type Reporter interface {
	Report(err error)
	Name() string
}

// Report reports an error to all reporters
func Report(err error) {
	g := new(errgroup.Group)
	for _, reporter := range reporters {
		g.Go(reporter.Report)
	}
	err := g.Wait()
	if err != nil {
		panic(err)
	}
}
