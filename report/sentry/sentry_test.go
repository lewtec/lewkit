package sentry

import (
	"context"
	"io"
	"testing"
	"time"

	sdk "github.com/getsentry/sentry-go"
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
	if err != nil {
		t.Fatal(err)
	}
	return &Reporter{hub: sdk.NewHub(client, sdk.NewScope())}
}

func TestParseRejectsBadDSN(t *testing.T) {
	t.Parallel()
	var r Reporter
	if err := r.Parse("not-a-dsn"); err == nil {
		t.Fatal("Parse(not-a-dsn) = nil error")
	}
}

func TestParseEmpty(t *testing.T) {
	t.Parallel()
	var r Reporter
	if err := r.Parse(""); err != nil {
		t.Fatal(err)
	}
	if err := r.Setup(); err != nil {
		t.Fatal(err)
	}
}

func TestParse(t *testing.T) {
	t.Parallel()
	var r Reporter
	if err := r.Parse("https://public@example.com/1"); err != nil {
		t.Fatal(err)
	}
	if got := r.Value(); got != "https://public@example.com/1" {
		t.Fatalf("Value() = %q", got)
	}
}

func TestReport(t *testing.T) {
	t.Parallel()
	tr := &captureTransport{}
	r := reporterWithTransport(t, tr)

	if err := r.Report(nil); err != nil {
		t.Fatalf("Report(nil) = %v", err)
	}
	if n := len(tr.events); n != 0 {
		t.Fatalf("Report(nil) sent %d events", n)
	}

	if err := r.Report(io.EOF); err != nil {
		t.Fatalf("Report(EOF) = %v", err)
	}
	if n := len(tr.events); n != 1 {
		t.Fatalf("Report(EOF) sent %d events, want 1", n)
	}
	got := exceptionValue(tr.events[0])
	if got != io.EOF.Error() {
		t.Fatalf("event exception = %q, want %q", got, io.EOF.Error())
	}
}

func exceptionValue(e *sdk.Event) string {
	if len(e.Exception) == 0 {
		return e.Message
	}
	return e.Exception[0].Value
}
