// Package sentry is the Sentry [github.com/lewtec/lewkit/report.Reporter].
package sentry

import (
	"errors"
	"fmt"

	sdk "github.com/getsentry/sentry-go"

	"github.com/lewtec/lewkit/report"
)

var _ report.Reporter = (*Reporter)(nil)

// Reporter sends errors to Sentry.
type Reporter struct {
	hub *sdk.Hub
}

// New builds a reporter for dsn.
// An empty dsn uses SENTRY_DSN; if that is also empty the client is disabled.
func New(dsn string) (*Reporter, error) {
	client, err := sdk.NewClient(sdk.ClientOptions{Dsn: dsn})
	if err != nil {
		return nil, fmt.Errorf("sentry client: %w", err)
	}
	return &Reporter{hub: sdk.NewHub(client, sdk.NewScope())}, nil
}

// Report captures err and waits for delivery.
func (r *Reporter) Report(err error) error {
	if err == nil {
		return nil
	}
	r.hub.CaptureException(err)
	if !r.hub.Flush(sdk.DefaultFlushTimeout) {
		return errFlushTimeout
	}
	return nil
}

var errFlushTimeout = errors.New("sentry flush timed out")
