package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTickerRendersCount(t *testing.T) {
	var count atomic.Int64
	count.Store(7)
	srv := httptest.NewServer(newTicker(&count))
	defer srv.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get(srv.URL + "/")
	require.NoError(t, err)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), `class="count">7<`)
	require.Equal(t, "no-store", res.Header.Get("Cache-Control"))
}
