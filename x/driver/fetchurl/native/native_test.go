package native

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lewtec/lewkit/x/driver/fetchurl"
	_ "github.com/lewtec/lewkit/x/driver/httpclient/native"

	"github.com/stretchr/testify/require"
)

func TestFetchChecksHash(t *testing.T) {
	t.Setenv("FETCHURL_SERVER", "")
	body := []byte("hello")
	sum := sha256.Sum256(body)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Write(body)
	}))
	t.Cleanup(server.Close)

	fetcher, err := factory{}.New(t.Context())
	require.NoError(t, err)
	var output bytes.Buffer
	err = fetcher.Fetch(t.Context(), fetchurl.FetchOptions{
		URLs: []string{server.URL},
		Algo: "sha256",
		Hash: hex.EncodeToString(sum[:]),
		Out:  &output,
	})
	require.NoError(t, err)
	require.Equal(t, "hello", output.String())

	output.Reset()
	err = fetcher.Fetch(t.Context(), fetchurl.FetchOptions{
		URLs: []string{server.URL},
		Algo: "sha256",
		Hash: "deadbeef",
		Out:  &output,
	})
	require.Error(t, err)
}

func TestFetchRunsConfigureRequest(t *testing.T) {
	t.Setenv("FETCHURL_SERVER", "")
	body := []byte("hello")
	sum := sha256.Sum256(body)
	var sawToken bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		sawToken = request.Header.Get("Authorization") == "Bearer test-token"
		writer.Write(body)
	}))
	t.Cleanup(server.Close)

	fetcher, err := factory{}.New(t.Context())
	require.NoError(t, err)
	var output bytes.Buffer
	err = fetcher.Fetch(t.Context(), fetchurl.FetchOptions{
		URLs: []string{server.URL},
		Algo: "sha256",
		Hash: hex.EncodeToString(sum[:]),
		Out:  &output,
		ConfigureRequest: func(request *http.Request) {
			request.Header.Set("Authorization", "Bearer test-token")
		},
	})
	require.NoError(t, err)
	require.True(t, sawToken)
}

func TestFetchDoesNotRewriteRedirects(t *testing.T) {
	t.Setenv("FETCHURL_SERVER", "")
	body := []byte("payload")
	sum := sha256.Sum256(body)
	var hits int
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		hits++
		if request.URL.Path == "/start" {
			http.Redirect(writer, request, server.URL+"/file", http.StatusFound)
			return
		}
		writer.Write(body)
	}))
	t.Cleanup(server.Close)

	fetcher, err := factory{}.New(t.Context())
	require.NoError(t, err)
	var output bytes.Buffer
	err = fetcher.Fetch(t.Context(), fetchurl.FetchOptions{
		URLs: []string{server.URL + "/start"},
		Algo: "sha256",
		Hash: hex.EncodeToString(sum[:]),
		Out:  &output,
		ConfigureRequest: func(request *http.Request) {
			request.URL.Path = "/start"
			request.Header.Set("Authorization", "Bearer test-token")
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, hits)
	require.Equal(t, "payload", output.String())
}
