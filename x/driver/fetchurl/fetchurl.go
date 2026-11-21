// Package fetchurl is the fetchurl protocol client.
//
// The native driver uses [github.com/fetchurl/fetchurl]. Fetch requires a hash
// and at least one source URL. FETCHURL_SERVER is tried first, then the source
// URLs. [FetchOptions.ConfigureRequest] runs on each request that client sends.
package fetchurl

import (
	"context"
	"errors"
	"io"
	"net/http"
)

var (
	// ErrNoURLs is returned when Fetch is given no URLs.
	ErrNoURLs = errors.New("no URLs provided")
	// ErrNoOutputWriter is returned when Fetch is given no writer.
	ErrNoOutputWriter = errors.New("no output writer provided")
)

// StatusError is a non-OK HTTP response.
type StatusError struct {
	URL    string
	Status string
	Code   int
}

func (err *StatusError) Error() string {
	return "GET " + err.URL + ": " + err.Status
}

// FetchOptions is one fetchurl download.
// Algo and Hash are required by the protocol. ConfigureRequest runs on each
// request before it is sent, so a caller can attach a GitHub token.
type FetchOptions struct {
	URLs             []string
	Algo             string
	Hash             string
	Out              io.Writer
	ConfigureRequest func(*http.Request)
}

// Driver downloads bytes and checks a hash when one is set.
type Driver interface {
	Fetch(ctx context.Context, options FetchOptions) error
}
