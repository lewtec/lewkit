package host

import (
	"path/filepath"
	"testing"
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
	if err != nil {
		t.Fatal(err)
	}
	if got.Data != "/data" || got.Cache != "/cache" || got.Config != "/config" {
		t.Fatalf("%+v", got)
	}
	if got.Inbox != filepath.Join("/cache", "inbox") {
		t.Fatalf("Inbox = %q", got.Inbox)
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
	if err != nil {
		t.Fatal(err)
	}
	wantData := filepath.Join("/home/u", ".local", "share", "br.tec.lew.counter")
	if got.Data != wantData {
		t.Fatalf("Data = %q", got.Data)
	}
	if got.Cache != filepath.Join("/cache", "br.tec.lew.counter") {
		t.Fatalf("Cache = %q", got.Cache)
	}
}

func TestDataHome_ByGOOS(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	home := func() (string, error) { return "/home/u", nil }

	got, err := dataHome("linux", getenv, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("/home/u", ".local", "share") {
		t.Fatalf("linux: %q", got)
	}
	got, err = dataHome("darwin", getenv, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("/home/u", "Library", "Application Support") {
		t.Fatalf("darwin: %q", got)
	}
	env["LOCALAPPDATA"] = filepath.Join("C:", "Users", "u", "AppData", "Local")
	got, err = dataHome("windows", getenv, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != env["LOCALAPPDATA"] {
		t.Fatalf("windows: %q", got)
	}
}
