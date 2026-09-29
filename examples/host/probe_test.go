package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lewtec/lewkit/x/http/asset/sakuracss"
	"github.com/stretchr/testify/require"
)

func TestProbeNames(t *testing.T) {
	t.Setenv("LEWKIT_APP_ID", "br.tec.lew.host")
	dir := t.TempDir()
	t.Setenv("LEWKIT_DATA_DIR", dir)
	t.Setenv("LEWKIT_CACHE_DIR", dir)
	t.Setenv("LEWKIT_CONFIG_DIR", dir)

	rows := probe(t.Context(), "br.tec.lew.host")
	got := map[string]string{}
	for _, row := range rows {
		got[row.Name] = row.Text
	}
	for _, name := range []string{"battery", "brightness", "volume", "mute", "sink", "screen", "night", "dirs"} {
		text, ok := got[name]
		require.True(t, ok, name)
		require.NotEmpty(t, text, name)
	}
}

func TestActUnknown(t *testing.T) {
	err := (action{Name: "nope"}).run(t.Context())
	require.ErrorIs(t, err, errUnknown)
}

func TestPageListsProbe(t *testing.T) {
	t.Setenv("LEWKIT_APP_ID", "br.tec.lew.host")
	dir := t.TempDir()
	t.Setenv("LEWKIT_DATA_DIR", dir)
	t.Setenv("LEWKIT_CACHE_DIR", dir)
	t.Setenv("LEWKIT_CONFIG_DIR", dir)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	newMux().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "battery")
	require.Contains(t, body, "Brightness up")
	require.Contains(t, body, "Copy text")
	require.NotContains(t, body, "Volume up")
	require.NotContains(t, body, "Copy image")
	require.NotContains(t, body, "Screen off")
	require.NotContains(t, body, "Shutdown")
	require.Contains(t, body, sakuracss.Path)

	style := httptest.NewRecorder()
	newMux().ServeHTTP(style, httptest.NewRequest(http.MethodGet, sakuracss.Path, nil))
	require.Equal(t, http.StatusOK, style.Code)
	require.Contains(t, style.Body.String(), "Sakura.css")
}
