package httpclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lewtec/lewkit/x/taskgroup"

	"github.com/stretchr/testify/require"
)

func TestByteProgress(t *testing.T) {
	require.Equal(t, "512.0 KiB / 2.0 MiB", byteProgress(512*1024, 2*1024*1024))
}

func TestHumanBytes(t *testing.T) {
	require.Equal(t, "0 B", humanBytes(0))
	require.Equal(t, "0 B", humanBytes(-1))
	require.Equal(t, "500 B", humanBytes(500))
	require.Equal(t, "1.0 KiB", humanBytes(1024))
	require.Equal(t, "1.5 KiB", humanBytes(1536))
	require.Equal(t, "1.0 MiB", humanBytes(1024*1024))
}

func TestTaskName(t *testing.T) {
	require.Equal(t, "bundle.tar.gz", taskName(requestWithURL(t, "https://cdn.example.com/path/bundle.tar.gz")))
	require.Equal(t, "github release", taskName(requestWithURL(t, "https://api.github.com/repos/o/r/releases/assets/12345")))
	require.Equal(t, "github", taskName(requestWithURL(t, "https://objects.githubusercontent.com/github-production-release-asset-2e65be/abc")))
	require.Equal(t, "example.com", taskName(requestWithURL(t, "https://example.com/api/v1/items/99")))
	require.Equal(t, "example.com", taskName(requestWithURL(t, "https://example.com/")))
}

func TestTaskNameWithLabel(t *testing.T) {
	request := requestWithURL(t, "https://api.github.com/repos/o/r/releases/assets/1")
	request = request.WithContext(WithTaskLabel(request.Context(), "tool@1.2.3"))
	require.Equal(t, "tool@1.2.3", taskName(request))
}

func TestWithProgressRegistersInternetTask(t *testing.T) {
	payload := bytesOf('x', 32)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Length", "32")
		writer.Write(payload)
	}))
	t.Cleanup(server.Close)

	session, ctx := taskgroup.New(t.Context(), taskgroup.DefaultLimits())
	t.Cleanup(func() { _ = session.Wait() })

	client := &http.Client{Transport: WithProgress(http.DefaultTransport)}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/bundle.tar.gz", nil)
	require.NoError(t, err)
	response, err := client.Do(request)
	require.NoError(t, err)
	_, err = io.Copy(io.Discard, response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.NotZero(t, session.Latest("bundle.tar.gz"))
	require.NoError(t, session.Wait())
}

func requestWithURL(t *testing.T, raw string) *http.Request {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, raw, nil)
	require.NoError(t, err)
	return request
}

func bytesOf(value byte, count int) []byte {
	buffer := make([]byte, count)
	for i := range buffer {
		buffer[i] = value
	}
	return buffer
}
