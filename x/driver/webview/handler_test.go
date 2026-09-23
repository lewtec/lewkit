package webview

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDispatchStreams(t *testing.T) {
	release := make(chan struct{})
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var payload strings.Builder
		_, _ = io.Copy(&payload, request.Body)
		if payload.String() != "n=1" {
			return
		}
		response.Header().Set("Content-Type", "text/plain")
		response.WriteHeader(http.StatusCreated)
		_, _ = response.Write([]byte("ok"))
		<-release
	})
	status, header, body, err := Dispatch(handler, http.MethodPost, "app://view1/count?n=1", nil, strings.NewReader("n=1"))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, status)
	require.Equal(t, "text/plain", header.Get("Content-Type"))
	chunk := make([]byte, 2)
	count, err := body.Read(chunk)
	require.NoError(t, err)
	require.Equal(t, "ok", string(chunk[:count]))
	close(release)
	_, err = io.Copy(io.Discard, body)
	require.NoError(t, err)
	require.NoError(t, body.Close())
}

func TestValidateHandler(t *testing.T) {
	require.NoError(t, Config{Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}.Validate())
}
