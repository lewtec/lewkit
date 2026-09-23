package native

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/httpclient"
)

type factory struct{}

func (factory) ID() string   { return "httpclient_native" }
func (factory) Name() string { return "Native HTTP" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(context.Context) (httpclient.Driver, error) {
	return clientDriver{client: &http.Client{Transport: httpclient.WithProgress(loggingTransport{base: http.DefaultTransport})}}, nil
}

type clientDriver struct {
	client *http.Client
}

func (driver clientDriver) Client() *http.Client { return driver.client }

type loggingTransport struct {
	base http.RoundTripper
}

func (transport loggingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	slog.DebugContext(request.Context(), "http request", "url", request.URL.String())
	base := transport.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(request)
}

var _ driver.DriverFactory[httpclient.Driver] = factory{}
var _ driver.Weighter = factory{}
