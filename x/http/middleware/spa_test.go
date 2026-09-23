package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func doRequest(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, nil)
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestSPAFile(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{
		"hello.txt": {Data: []byte("hi")},
	}
	recorder := doRequest(SPA(filesystem, nil), http.MethodGet, "/hello.txt")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "hi", recorder.Body.String())
	require.Equal(t, "max-age=5", recorder.Header().Get("Cache-Control"))
}

func TestSPANeverLists(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{
		"photo.jpg": {Data: []byte("img")},
	}
	recorder := doRequest(SPA(filesystem, nil), http.MethodGet, "/")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "photo.jpg")
}

func TestSPADirectoryIndex(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{
		"app/index.html": {Data: []byte("spa-app")},
	}
	redirect := doRequest(SPA(filesystem, nil), http.MethodGet, "/app")
	require.Equal(t, http.StatusPermanentRedirect, redirect.Code)
	require.Equal(t, "/app/", redirect.Header().Get("Location"))

	recorder := doRequest(SPA(filesystem, nil), http.MethodGet, "/app/")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "spa-app", recorder.Body.String())
}

func TestSPAMissChain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		files      fstest.MapFS
		target     string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "site 404 wins",
			files:      fstest.MapFS{"404.html": {Data: []byte("site-404")}, "index.html": {Data: []byte("site-index")}},
			target:     "/nope",
			wantStatus: http.StatusNotFound,
			wantBody:   "site-404",
		},
		{
			name:       "site index when no 404",
			files:      fstest.MapFS{"index.html": {Data: []byte("site-index")}},
			target:     "/nope",
			wantStatus: http.StatusOK,
			wantBody:   "site-index",
		},
		{
			name:       "next when neither",
			files:      fstest.MapFS{"other.txt": {Data: []byte("x")}},
			target:     "/nope",
			wantStatus: http.StatusTeapot,
			wantBody:   "fallback",
		},
		{
			name:       "no walk up",
			files:      fstest.MapFS{"app/index.html": {Data: []byte("nested")}},
			target:     "/app/user/42",
			wantStatus: http.StatusTeapot,
			wantBody:   "fallback",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			next := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(http.StatusTeapot)
				_, err := io.WriteString(response, "fallback")
				require.NoError(t, err)
			})
			recorder := doRequest(SPA(tt.files, next), http.MethodGet, tt.target)
			require.Equal(t, tt.wantStatus, recorder.Code)
			require.Contains(t, recorder.Body.String(), tt.wantBody)
			if tt.name == "no walk up" {
				require.NotContains(t, recorder.Body.String(), "nested")
			}
		})
	}
}

func TestSPASite404KeepsStatus(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{
		"404.html": {Data: []byte("<p>missing</p>")},
	}
	recorder := doRequest(SPA(filesystem, nil), http.MethodGet, "/missing")
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/html")
	require.Equal(t, "<p>missing</p>", recorder.Body.String())
	require.Equal(t, "max-age=5", recorder.Header().Get("Cache-Control"))
}

func TestSPAHead(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{
		"hello.txt": {Data: []byte("hi")},
		"404.html":  {Data: []byte("missing-page")},
	}
	file := doRequest(SPA(filesystem, nil), http.MethodHead, "/hello.txt")
	require.Equal(t, http.StatusOK, file.Code)
	require.Empty(t, file.Body.String())

	missing := doRequest(SPA(filesystem, nil), http.MethodHead, "/nope")
	require.Equal(t, http.StatusNotFound, missing.Code)
	require.Empty(t, missing.Body.String())
	require.Equal(t, "12", missing.Header().Get("Content-Length"))
}

func TestSPAMethod(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{}
	rejected := doRequest(SPA(filesystem, nil), http.MethodPost, "/")
	require.Equal(t, http.StatusMethodNotAllowed, rejected.Code)
	require.Equal(t, "GET, HEAD", rejected.Header().Get("Allow"))

	next := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusAccepted)
	})
	accepted := doRequest(SPA(filesystem, next), http.MethodPost, "/api")
	require.Equal(t, http.StatusAccepted, accepted.Code)
}

func TestSPAQueryString(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{
		"hello.txt": {Data: []byte("hi")},
	}
	recorder := doRequest(SPA(filesystem, nil), http.MethodGet, "/hello.txt?download=1")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "hi", recorder.Body.String())
}

func TestSPAFileWinsOverIndex(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{
		"index.html": {Data: []byte("site")},
		"app.txt":    {Data: []byte("file")},
	}
	recorder := doRequest(SPA(filesystem, nil), http.MethodGet, "/app.txt")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "file", recorder.Body.String())
}

func TestSPAEncodedName(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{
		"hello world.txt": {Data: []byte("hi")},
	}
	recorder := doRequest(SPA(filesystem, nil), http.MethodGet, "/hello%20world.txt")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "hi", recorder.Body.String())
}

func TestSPARootIndex(t *testing.T) {
	t.Parallel()
	filesystem := fstest.MapFS{
		"index.html": {Data: []byte("home")},
	}
	recorder := doRequest(SPA(filesystem, nil), http.MethodGet, "/")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "home", recorder.Body.String())
	require.False(t, strings.Contains(recorder.Body.String(), "<table"))
}
