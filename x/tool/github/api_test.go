package github

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type errReader struct{ err error }

func (reader errReader) Read([]byte) (int, error) { return 0, reader.err }

func TestAPIErrorFromResponse(t *testing.T) {
	const requestURL = "https://api.github.com/repos/o/r/releases"

	t.Run("rate limit when body readable", func(t *testing.T) {
		response := &http.Response{
			Status:     "403 Forbidden",
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(`{"message":"API rate limit exceeded"}`)),
		}
		err := apiErrorFromResponse(requestURL, response)
		if !errors.Is(err, ErrAPIError) || !errors.Is(err, ErrAPIRateLimit) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("read failure wraps the read error", func(t *testing.T) {
		readErr := errors.New("boom")
		response := &http.Response{
			Status:     "502 Bad Gateway",
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(errReader{err: readErr}),
		}
		err := apiErrorFromResponse(requestURL, response)
		if !errors.Is(err, ErrAPIError) || !errors.Is(err, readErr) {
			t.Fatalf("got %v", err)
		}
		if errors.Is(err, ErrAPIRateLimit) {
			t.Fatalf("rate limit without a body: %v", err)
		}
	})
}
