package host

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPaths_EnvWins(t *testing.T) {
	got, err := paths(
		"br.tec.lew.counter",
		"linux",
		func(k string) string {
			switch k {
			case envData:
				return "/data"
			case envCache:
				return "/cache"
			case envConfig:
				return "/config"
			default:
				return ""
			}
		},
		func() (string, error) { return "/home/u", nil },
		func() (string, error) { return "/unused-cache", nil },
		func() (string, error) { return "/unused-config", nil },
	)
	require.NoError(t, err)
	if got.Data != "/data" || got.Cache != "/cache" || got.Config != "/config" {
		require.FailNowf(t, "", "%+v", got)
	}
	if got.Inbox != filepath.Join("/cache", "inbox") {
		require.FailNowf(t, "", "Inbox = %q", got.Inbox)
	}
}

func TestPaths_AppIDUnderHome(t *testing.T) {
	got, err := paths(
		"br.tec.lew.counter",
		"linux",
		func(string) string { return "" },
		func() (string, error) { return "/home/u", nil },
		func() (string, error) { return "/cache", nil },
		func() (string, error) { return "/config", nil },
	)
	require.NoError(t, err)
	wantData := filepath.Join("/home/u", ".local", "share", "br.tec.lew.counter")
	if got.Data != wantData {
		require.FailNowf(t, "", "Data = %q", got.Data)
	}
	if got.Cache != filepath.Join("/cache", "br.tec.lew.counter") {
		require.FailNowf(t, "", "Cache = %q", got.Cache)
	}
}

func TestDataHome_ByGOOS(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	home := func() (string, error) { return "/home/u", nil }

	got, err := dataHome("linux", getenv, home)
	require.NoError(t, err)
	if got != filepath.Join("/home/u", ".local", "share") {
		require.FailNowf(t, "", "linux: %q", got)
	}
	got, err = dataHome("darwin", getenv, home)
	require.NoError(t, err)
	if got != filepath.Join("/home/u", "Library", "Application Support") {
		require.FailNowf(t, "", "darwin: %q", got)
	}
	env["LOCALAPPDATA"] = filepath.Join("C:", "Users", "u", "AppData", "Local")
	got, err = dataHome("windows", getenv, home)
	require.NoError(t, err)
	if got != env["LOCALAPPDATA"] {
		require.FailNowf(t, "", "windows: %q", got)
	}
}
