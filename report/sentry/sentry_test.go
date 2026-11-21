package sentry

import (
	"context"
	"io"
	"testing"
	"time"

	sdk "github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/require"
)

type captureTransport struct {
	events []*sdk.Event
}

func (t *captureTransport) Configure(sdk.ClientOptions) {}
func (t *captureTransport) SendEvent(e *sdk.Event)      { t.events = append(t.events, e) }
func (t *captureTransport) Flush(time.Duration) bool    { return true }
func (t *captureTransport) FlushWithContext(context.Context) bool {
	return true
}
func (t *captureTransport) Close() {}

func reporterWithTransport(t *testing.T, tr sdk.Transport) *Reporter {
	t.Helper()
	client, err := sdk.NewClient(sdk.ClientOptions{
		Dsn:       "https://public@example.com/1",
		Transport: tr,
	})
	require.NoError(t, err)
	return &Reporter{hub: sdk.NewHub(client, sdk.NewScope())}
}

func TestNewRejectsBadDSN(t *testing.T) {
	t.Parallel()
	_, err := New("not-a-dsn")
	require.Error(t, err)
}

func TestReport(t *testing.T) {
	t.Parallel()
	tr := &captureTransport{}
	r := reporterWithTransport(t, tr)

	require.NoError(t, r.Report(nil))
	require.Empty(t, tr.events)

	require.NoError(t, r.Report(io.EOF))
	require.Len(t, tr.events, 1)
	require.Equal(t, io.EOF.Error(), exceptionValue(tr.events[0]))
}

func exceptionValue(e *sdk.Event) string {
	if len(e.Exception) == 0 {
		return e.Message
	}
	return e.Exception[0].Value
}
