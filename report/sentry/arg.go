package sentry

import "github.com/lewtec/lewkit/report"

// Arg is a Sentry DSN for x/cmd. Parse builds the reporter New would return.
type Arg struct {
	*Reporter
}

func (a *Arg) Parse(arg string) error {
	if arg == "" {
		a.Reporter = nil
		return nil
	}
	r, err := New(arg)
	if err != nil {
		return err
	}
	a.Reporter = r
	return nil
}

// Setup registers the parsed reporter.
func (a *Arg) Setup() error {
	if a.Reporter == nil {
		return nil
	}
	report.RegisterReporter(a.Reporter)
	return nil
}
