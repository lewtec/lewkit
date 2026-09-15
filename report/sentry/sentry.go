// Package sentry is the Sentry [github.com/lewtec/lewkit/report.Reporter].
package sentry

import (
	"errors"
	"fmt"

	sdk "github.com/getsentry/sentry-go"

	"github.com/lewtec/lewkit/report"
)

var _ report.Reporter = (*Reporter)(nil)

// Reporter sends errors to Sentry. It is also an x/cmd arg for a DSN.
type Reporter struct {
	dsn string
	hub *sdk.Hub
}

func (r *Reporter) Parse(arg string) error {
	if arg == "" {
		r.dsn = ""
		r.hub = nil
		return nil
	}
	client, err := sdk.NewClient(sdk.ClientOptions{Dsn: arg})
	if err != nil {
		return fmt.Errorf("sentry client: %w", err)
	}
	r.dsn = arg
	r.hub = sdk.NewHub(client, sdk.NewScope())
	return nil
}

func (r Reporter) Value() string {
	return r.dsn
}

// Setup registers the reporter when a DSN was parsed.
func (r *Reporter) Setup() error {
	if r.hub == nil {
		return nil
	}
	report.RegisterReporter(r)
	return nil
}

// Report captures err and waits for delivery.
func (r *Reporter) Report(err error) error {
	if err == nil || r.hub == nil {
		return nil
	}
	r.hub.CaptureException(err)
	if !r.hub.Flush(sdk.DefaultFlushTimeout) {
		return errFlushTimeout
	}
	return nil
}

var errFlushTimeout = errors.New("sentry flush timed out")
