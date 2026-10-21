package site

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestZipRangeAndSPA(t *testing.T) {
	body := zipBytes(t, map[string]string{
		"site/index.html": "<h1>home</h1>",
		"site/a.mp3":      "0123456789",
	})
	handler, err := Zip(t.Context(), bytes.NewReader(body))
	require.NoError(t, err)

	rangeRequest := httptest.NewRequest(http.MethodGet, "/a.mp3", nil)
	rangeRequest.Header.Set("Range", "bytes=0-3")
	rangeResponse := httptest.NewRecorder()
	handler.ServeHTTP(rangeResponse, rangeRequest)
	require.Equal(t, http.StatusPartialContent, rangeResponse.Code)
	require.Equal(t, "0123", rangeResponse.Body.String())

	miss := httptest.NewRequest(http.MethodGet, "/missing", nil)
	missResponse := httptest.NewRecorder()
	handler.ServeHTTP(missResponse, miss)
	require.Equal(t, http.StatusOK, missResponse.Code)
	require.Equal(t, "<h1>home</h1>", missResponse.Body.String())
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, body := range files {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		file, err := writer.CreateHeader(header)
		require.NoError(t, err)
		_, err = file.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buf.Bytes()
}
