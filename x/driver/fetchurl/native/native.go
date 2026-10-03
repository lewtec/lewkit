package native

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
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
	// The HTTP client already promotes each request to an Internet task.
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
	fetcher := downloader.fetcher
	if next, skipped := withoutUnresolvableServers(ctx, fetcher); len(skipped) > 0 {
		fetcher = next
	}
	err := fetcher.Fetch(ctx, sdk.FetchOptions{
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

// withoutUnresolvableServers drops servers whose host does not resolve.
// A failed dial is a taskgroup error and cancels the session, so the source
// URLs would not be tried. The lookup miss is logged and the caller carries on.
func withoutUnresolvableServers(ctx context.Context, fetcher *sdk.Fetcher) (*sdk.Fetcher, []string) {
	if fetcher == nil || len(fetcher.Servers) == 0 || ctx.Err() != nil {
		return fetcher, nil
	}
	kept := make([]string, 0, len(fetcher.Servers))
	var skipped []string
	var miss error
	for _, server := range fetcher.Servers {
		err := resolutionMiss(ctx, server)
		if err != nil {
			skipped = append(skipped, server)
			miss = err
			continue
		}
		kept = append(kept, server)
	}
	if len(skipped) == 0 {
		return fetcher, nil
	}
	slog.WarnContext(ctx, "fetchurl server host did not resolve", "server", skipped, "error", miss)
	copied := *fetcher
	copied.Servers = kept
	return &copied, skipped
}

func resolutionMiss(ctx context.Context, server string) error {
	if ctx.Err() != nil {
		return nil
	}
	parsed, err := url.Parse(server)
	if err != nil {
		return nil
	}
	host := parsed.Hostname()
	if host == "" || net.ParseIP(host) != nil {
		return nil
	}
	_, err = net.DefaultResolver.LookupHost(ctx, host)
	if err == nil || ctx.Err() != nil {
		return nil
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		return err
	}
	return nil
}

type hookTransport struct {
	base http.RoundTripper
}

func (transport hookTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	// Configure runs once, on the first request. A redirect keeps its own URL.
	// Reapplying the hook sends GitHub asset downloads back to the API and loops.
	if request.Response == nil {
		if configure, ok := request.Context().Value(configureKey{}).(func(*http.Request)); ok && configure != nil {
			configure(request)
		}
	}
	base := transport.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(request)
}

var _ driver.DriverFactory[fetchurl.Driver] = factory{}
var _ driver.Weighter = factory{}
