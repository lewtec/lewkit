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
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err = fetcher.Fetch(t.Context(), fetchurl.FetchOptions{
		URLs: []string{server.URL},
		Algo: "sha256",
		Hash: hex.EncodeToString(sum[:]),
		Out:  &output,
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != "hello" {
		t.Fatalf("body = %q", output.String())
	}

	output.Reset()
	err = fetcher.Fetch(t.Context(), fetchurl.FetchOptions{
		URLs: []string{server.URL},
		Algo: "sha256",
		Hash: "deadbeef",
		Out:  &output,
	})
	if err == nil {
		t.Fatal("expected hash mismatch")
	}
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
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if !sawToken {
		t.Fatal("ConfigureRequest did not run")
	}
}
