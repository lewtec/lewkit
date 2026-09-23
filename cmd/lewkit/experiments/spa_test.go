package experiments

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lewtec/lewkit/x/http/asset/sakuracss"
	"github.com/stretchr/testify/require"
)

func TestSPAPage(t *testing.T) {
	handler := spaHandler(t.Context())
	for _, path := range []string{"/", "/about"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		handler.ServeHTTP(response, request)
		require.Equal(t, http.StatusOK, response.Code, path)
		require.Contains(t, response.Body.String(), "templ page")
		require.Contains(t, response.Body.String(), sakuracss.Path)
	}
	style := httptest.NewRecorder()
	handler.ServeHTTP(style, httptest.NewRequest(http.MethodGet, sakuracss.Path, nil))
	require.Equal(t, http.StatusOK, style.Code)
	require.Contains(t, style.Body.String(), "Sakura.css")
}
