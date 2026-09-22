// Package fetchurl is the hash-checked download capability.
//
// [Driver.Fetch] tries URLs in order through the selected [httpclient.Driver].
// A hash is checked when Algo and Hash are set. An empty hash stores the body as-is.
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

// FetchOptions is one download.
// URLs are tried in order. ConfigureRequest runs on each request before it is sent.
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
