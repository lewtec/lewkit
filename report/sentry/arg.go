package sentry

import "github.com/lewtec/lewkit/report"

// Arg is a Sentry DSN for x/cmd. Parse builds the reporter New would return.
type Arg struct {
	value string
	rep   *Reporter
}

func (a *Arg) Parse(arg string) error {
	if arg == "" {
		a.value = ""
		a.rep = nil
		return nil
	}
	r, err := New(arg)
	if err != nil {
		return err
	}
	a.value = arg
	a.rep = r
	return nil
}

func (a Arg) Value() string {
	return a.value
}

// Setup registers the parsed reporter.
func (a *Arg) Setup() error {
	if a.rep == nil {
		return nil
	}
	report.RegisterReporter(a.rep)
	return nil
}
