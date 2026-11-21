package asset

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterRejects(t *testing.T) {
	t.Parallel()
	set := &registry{}
	err := set.register(File{Name: "a/b.js", ContentType: "text/javascript", Body: []byte("x")})
	require.ErrorIs(t, err, ErrName)
	err = set.register(File{Name: "a.js", Body: nil})
	require.ErrorIs(t, err, ErrEmpty)
	require.NoError(t, set.register(File{Name: "a.js", Body: []byte("x")}))
	err = set.register(File{Name: "a.js", Body: []byte("y")})
	require.ErrorIs(t, err, ErrExist)
}

func TestMountFallthrough(t *testing.T) {
	t.Parallel()
	next := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusTeapot)
		_, err := io.WriteString(response, "next")
		require.NoError(t, err)
	})
	handler := assetHandler{next: next}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/app", nil))
	require.Equal(t, http.StatusTeapot, recorder.Code)
	require.Equal(t, "next", recorder.Body.String())

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, Prefix+"missing.js", nil))
	require.Equal(t, http.StatusNotFound, missing.Code)
}
