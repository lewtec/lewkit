package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPageSaysItWorks(t *testing.T) {
	srv := httptest.NewServer(newPage())
	defer srv.Close()
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Get(srv.URL + "/")
	require.NoError(t, err)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, "it works!", string(body))
}
