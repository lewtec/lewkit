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

func TestDispatchFollowsSameViewRedirect(t *testing.T) {
	var paths []string
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		paths = append(paths, request.URL.Path)
		if request.URL.Path == "/go" {
			http.Redirect(response, request, "/", http.StatusSeeOther)
			return
		}
		if request.URL.Path == "/away" {
			http.Redirect(response, request, "https://example.com/docs", http.StatusSeeOther)
			return
		}
		_, _ = response.Write([]byte("page " + request.URL.Path))
	})

	status, header, body, err := Dispatch(handler, http.MethodPost, "app://view9/go", nil, strings.NewReader("op=inc"))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Empty(t, header.Get("Location"))
	text, err := io.ReadAll(body)
	require.NoError(t, err)
	require.Equal(t, "page /", string(text))
	require.Equal(t, []string{"/go", "/"}, paths)

	status, header, body, err = Dispatch(handler, http.MethodGet, "app://view9/away", nil, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusSeeOther, status)
	require.Equal(t, "https://example.com/docs", header.Get("Location"))
	require.NoError(t, body.Close())
}

func TestValidateHandler(t *testing.T) {
	require.NoError(t, Config{Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}.Validate())
}
