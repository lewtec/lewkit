package profile

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectoryWritesProfiles(t *testing.T) {
	temp := t.TempDir()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	p := Directory(temp)
	assert.Equal(t, temp, p.Directory())
	assert.Empty(t, p.Address())
	assert.NoError(t, p.Run(ctx))

	cpu, err := os.Stat(filepath.Join(temp, "cpu.prof"))
	require.NoError(t, err)
	assert.NotZero(t, cpu.Size())
	for _, prof := range pprof.Profiles() {
		filename := p.file(prof.Name())
		stat, err := os.Stat(filename)
		if assert.NoError(t, err, filename) {
			assert.NotZero(t, stat.Size(), filename)
		}
	}
}

func TestHandlerServesIndex(t *testing.T) {
	server := httptest.NewServer(Handler())
	t.Cleanup(func() { server.Close() })

	resp := getResponse(t, server.URL+"/debug/pprof/")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "heap")
	assert.Contains(t, string(body), "goroutine")
}

func TestHandlerServesHeap(t *testing.T) {
	server := httptest.NewServer(Handler())
	t.Cleanup(func() { server.Close() })

	resp := getResponse(t, server.URL+"/debug/pprof/heap")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func getResponse(t *testing.T, url string) *http.Response {
	t.Helper()
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(url)
	require.NoError(t, err)
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestAddressRunServes(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	p := Address(addr)
	assert.Equal(t, addr, p.Address())
	assert.Empty(t, p.Directory())
	errCh := make(chan error, 1)
	go func() { errCh <- p.Run(ctx) }()

	client := &http.Client{Timeout: 200 * time.Millisecond}
	url := "http://" + addr + "/debug/pprof/"
	require.Eventually(t, func() bool {
		resp, err := client.Get(url)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 2*time.Second, 10*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
}
