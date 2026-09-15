package cmd

import (
	"github.com/lewtec/lewkit/report"
	"github.com/lewtec/lewkit/report/sentry"
)

// SentryArg is a Sentry DSN. Empty leaves the report registry unchanged.
type SentryArg struct {
	Container[string]
}

func (s *SentryArg) Parse(arg string) error {
	s.value = arg
	return nil
}

// Setup registers a Sentry reporter when the DSN is set.
func (s SentryArg) Setup() error {
	if s.value == "" {
		return nil
	}
	r, err := sentry.New(s.value)
	if err != nil {
		return err
	}
	report.RegisterReporter(r)
	return nil
}

var (
	_ Parser      = (*SentryArg)(nil)
	_ Arg[string] = (*SentryArg)(nil)
)
