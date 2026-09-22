package native

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	sdk "github.com/fetchurl/fetchurl"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/fetchurl"
	"github.com/lewtec/lewkit/x/driver/httpclient"
)

type configureKey struct{}

type factory struct{}

func (factory) ID() string   { return "fetchurl_native" }
func (factory) Name() string { return "fetchurl" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(ctx context.Context) (fetchurl.Driver, error) {
	httpDriver, err := driver.Get[httpclient.Driver](ctx)
	if err != nil {
		return nil, fmt.Errorf("httpclient driver: %w", err)
	}
	client := httpDriver.Client()
	if client == nil {
		client = http.DefaultClient
	}
	wrapped := *client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	wrapped.Transport = hookTransport{base: base}
	return downloader{fetcher: sdk.NewFetcher(&wrapped)}, nil
}

type downloader struct {
	fetcher *sdk.Fetcher
}

func (downloader downloader) Fetch(ctx context.Context, options fetchurl.FetchOptions) error {
	if len(options.URLs) == 0 {
		return fetchurl.ErrNoURLs
	}
	if options.Out == nil {
		return fetchurl.ErrNoOutputWriter
	}
	if options.ConfigureRequest != nil {
		ctx = context.WithValue(ctx, configureKey{}, options.ConfigureRequest)
	}
	err := downloader.fetcher.Fetch(ctx, sdk.FetchOptions{
		Algo: options.Algo,
		Hash: options.Hash,
		URLs: options.URLs,
		Out:  options.Out,
	})
	if err == nil {
		return nil
	}
	var status *sdk.HTTPStatusError
	if errors.As(err, &status) {
		return fmt.Errorf("%w: %w", &fetchurl.StatusError{
			Code:   status.StatusCode,
			Status: strconv.Itoa(status.StatusCode),
		}, err)
	}
	return err
}

type hookTransport struct {
	base http.RoundTripper
}

func (transport hookTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if configure, ok := request.Context().Value(configureKey{}).(func(*http.Request)); ok && configure != nil {
		configure(request)
	}
	base := transport.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(request)
}

var _ driver.DriverFactory[fetchurl.Driver] = factory{}
var _ driver.Weighter = factory{}
