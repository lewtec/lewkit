package install

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveToolVersion(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	list := func(context.Context) ([]string, error) {
		return []string{"ruby-3.3.0", "ruby-3.2.0"}, nil
	}
	normalize := func(version string) string {
		version = strings.TrimSpace(version)
		version = strings.TrimPrefix(version, "ruby-")
		return version
	}

	t.Run("explicit version", func(t *testing.T) {
		t.Parallel()
		got, err := ResolveToolVersion(ctx, "ruby-3.2.0", normalize, list)
		require.NoError(t, err)
		require.Equal(t, "3.2.0", got)
	})

	t.Run("latest normalizes listed version", func(t *testing.T) {
		t.Parallel()
		got, err := ResolveToolVersion(ctx, "latest", normalize, list)
		require.NoError(t, err)
		require.Equal(t, "3.3.0", got)
	})

	t.Run("empty version treated as latest", func(t *testing.T) {
		t.Parallel()
		got, err := ResolveToolVersion(ctx, "", normalize, list)
		require.NoError(t, err)
		require.Equal(t, "3.3.0", got)
	})

	t.Run("no versions", func(t *testing.T) {
		t.Parallel()
		_, err := ResolveToolVersion(ctx, "latest", normalize, func(context.Context) ([]string, error) {
			return nil, nil
		})
		require.ErrorIs(t, err, ErrNoVersions)
	})
}

func TestSortVersionsDesc(t *testing.T) {
	t.Parallel()

	t.Run("empty yields ErrNoVersions", func(t *testing.T) {
		t.Parallel()
		_, err := SortVersionsDesc(nil)
		require.ErrorIs(t, err, ErrNoVersions)
		_, err = SortVersionsDesc([]string{})
		require.ErrorIs(t, err, ErrNoVersions)
	})

	t.Run("newest first", func(t *testing.T) {
		t.Parallel()
		got, err := SortVersionsDesc([]string{"1.2.0", "2.0.0", "1.10.0"})
		require.NoError(t, err)
		require.Equal(t, []string{"2.0.0", "1.10.0", "1.2.0"}, got)
	})
}

func TestHTTPGetHelpers(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	t.Run("getBytes and configure", func(t *testing.T) {
		t.Parallel()
		var gotAccept string
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			gotAccept = request.Header.Get("Accept")
			_, err := writer.Write([]byte("hello"))
			require.NoError(t, err)
		}))
		t.Cleanup(server.Close)

		body, err := GetBytes(ctx, server.URL, func(request *http.Request) {
			request.Header.Set("Accept", "text/plain")
		})
		require.NoError(t, err)
		require.Equal(t, "hello", string(body))
		require.Equal(t, "text/plain", gotAccept)
	})

	t.Run("getJSON", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, err := writer.Write([]byte(`{"version":"1.2.3"}`))
			require.NoError(t, err)
		}))
		t.Cleanup(server.Close)

		var destination struct {
			Version string `json:"version"`
		}
		require.NoError(t, GetJSON(ctx, server.URL, &destination))
		require.Equal(t, "1.2.3", destination.Version)
	})

	t.Run("non-OK", func(t *testing.T) {
		t.Parallel()
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			http.Error(writer, "nope", http.StatusNotFound)
		}))
		t.Cleanup(server.Close)

		_, err := GetBytes(ctx, server.URL)
		var status unexpectedHTTPStatusError
		require.ErrorAs(t, err, &status)
	})
}

func TestParseGNUHashFile(t *testing.T) {
	t.Parallel()
	got := ParseGNUHashFile([]byte("abc123  foo.tar.gz\n# comment\ndef456  bar.zip\nmalformed\n"))
	require.Equal(t, "abc123", got["foo.tar.gz"])
	require.Equal(t, "def456", got["bar.zip"])
	_, ok := got["malformed"]
	require.False(t, ok)
}
