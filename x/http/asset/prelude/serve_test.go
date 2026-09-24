package prelude

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/http/asset"
	"github.com/lewtec/lewkit/x/http/asset/htmx"
	"github.com/lewtec/lewkit/x/http/asset/jquery"
	"github.com/lewtec/lewkit/x/http/asset/lewtec_logo"
	"github.com/lewtec/lewkit/x/http/asset/sakuracss"
	"github.com/lewtec/lewkit/x/http/asset/tailwindcss"
	"github.com/stretchr/testify/require"
)

func TestPreludeServes(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusTeapot)
		_, err := io.WriteString(response, "next")
		require.NoError(t, err)
	})
	handler := asset.Mount(next)
	tests := []struct {
		path        string
		contentType string
		needle      string
	}{
		{path: htmx.Path, contentType: "text/javascript", needle: "htmx"},
		{path: jquery.Path, contentType: "text/javascript", needle: "jQuery"},
		{path: tailwindcss.Path, contentType: "text/javascript", needle: tailwindcss.Version},
		{path: sakuracss.Path, contentType: "text/css", needle: "Sakura.css"},
		{path: lewtec_logo.Path, contentType: "image/png", needle: "PNG"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, http.StatusOK, recorder.Code)
			require.Contains(t, recorder.Header().Get("Content-Type"), tt.contentType)
			require.Equal(t, "max-age=5", recorder.Header().Get("Cache-Control"))
			require.Contains(t, recorder.Body.String(), tt.needle)

			head := httptest.NewRecorder()
			handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, tt.path, nil))
			require.Equal(t, http.StatusOK, head.Code)
			require.Empty(t, head.Body.String())
		})
	}

	posted := httptest.NewRecorder()
	handler.ServeHTTP(posted, httptest.NewRequest(http.MethodPost, htmx.Path, nil))
	require.Equal(t, http.StatusMethodNotAllowed, posted.Code)
	require.Equal(t, "GET, HEAD", posted.Header().Get("Allow"))

	elsewhere := httptest.NewRecorder()
	handler.ServeHTTP(elsewhere, httptest.NewRequest(http.MethodGet, "/index.html", nil))
	require.Equal(t, http.StatusTeapot, elsewhere.Code)
	require.Equal(t, "next", elsewhere.Body.String())

	require.True(t, strings.HasPrefix(htmx.Path, asset.Prefix))
}
