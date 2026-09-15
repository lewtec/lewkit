package sentry

import "github.com/lewtec/lewkit/report"

// Arg is a Sentry DSN for x/cmd. Empty leaves the report registry unchanged.
type Arg struct {
	value string
}

func (a *Arg) Parse(arg string) error {
	a.value = arg
	return nil
}

func (a Arg) Value() string {
	return a.value
}

// Setup registers a Sentry reporter when the DSN is set.
func (a Arg) Setup() error {
	if a.value == "" {
		return nil
	}
	r, err := New(a.value)
	if err != nil {
		return err
	}
	report.RegisterReporter(r)
	return nil
}
