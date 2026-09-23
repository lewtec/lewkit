package experiments

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
	}
}
