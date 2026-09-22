package native

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/fetchurl"
	"github.com/lewtec/lewkit/x/driver/httpclient"
)

type factory struct{}

func (factory) ID() string   { return "fetchurl_native" }
func (factory) Name() string { return "Native fetch" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(ctx context.Context) (fetchurl.Driver, error) {
	httpDriver, err := driver.Get[httpclient.Driver](ctx)
	if err != nil {
		return nil, fmt.Errorf("httpclient driver: %w", err)
	}
	return downloader{client: httpDriver.Client()}, nil
}

type downloader struct {
	client *http.Client
}

func (downloader downloader) Fetch(ctx context.Context, options fetchurl.FetchOptions) error {
	if len(options.URLs) == 0 {
		return fetchurl.ErrNoURLs
	}
	if options.Out == nil {
		return fetchurl.ErrNoOutputWriter
	}
	client := downloader.client
	if client == nil {
		client = http.DefaultClient
	}
	var lastErr error
	for _, rawURL := range options.URLs {
		if strings.TrimSpace(rawURL) == "" {
			continue
		}
		err := fetchOne(ctx, client, rawURL, options)
		if err == nil {
			return nil
		}
		lastErr = err
	}
	if lastErr == nil {
		return fetchurl.ErrNoURLs
	}
	return lastErr
}

func fetchOne(ctx context.Context, client *http.Client, rawURL string, options fetchurl.FetchOptions) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	if options.ConfigureRequest != nil {
		options.ConfigureRequest(request)
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &fetchurl.StatusError{URL: rawURL, Status: response.Status, Code: response.StatusCode}
	}
	hasher, err := hasherFor(options.Algo, options.Hash)
	if err != nil {
		return err
	}
	writer := io.Writer(options.Out)
	if hasher != nil {
		writer = io.MultiWriter(options.Out, hasher)
	}
	if _, err := io.Copy(writer, response.Body); err != nil {
		return err
	}
	if hasher == nil {
		return nil
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(sum, options.Hash) {
		return fmt.Errorf("hash mismatch for %s", rawURL)
	}
	return nil
}

func hasherFor(algo, sum string) (hash.Hash, error) {
	if strings.TrimSpace(sum) == "" {
		return nil, nil
	}
	switch strings.ToLower(algo) {
	case "", "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported hash algorithm %q", algo)
	}
}

var _ driver.DriverFactory[fetchurl.Driver] = factory{}
var _ driver.Weighter = factory{}
