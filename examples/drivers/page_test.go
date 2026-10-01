package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/stretchr/testify/require"
)

type fixedBattery struct{}

func (fixedBattery) ID() string                               { return "battery_fixed" }
func (fixedBattery) Name() string                             { return "Fixed battery" }
func (fixedBattery) Weight() int                              { return 0 }
func (fixedBattery) CheckCompatibility(context.Context) error { return nil }
func (fixedBattery) New(context.Context) (battery.Driver, error) {
	return fixedBattery{}, nil
}
func (fixedBattery) BatteryStatus(context.Context) (battery.Status, error) {
	return battery.Discharging, nil
}
func (fixedBattery) BatteryLevel(context.Context) (int, error) { return 42, nil }

func init() { driver.Register[battery.Driver](fixedBattery{}) }

func TestHomeListsADriver(t *testing.T) {
	srv := httptest.NewServer(newPage(t.Context()))
	defer srv.Close()
	client := srv.Client()
	client.Timeout = 5 * time.Second
	res, err := client.Get(srv.URL + "/")
	require.NoError(t, err)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	text := string(body)
	require.Contains(t, text, "Open triangle")
	require.Contains(t, text, "<h2")
	require.Contains(t, text, `href="/driver/battery"`)
	require.NotContains(t, text, "location.href")
	require.NotContains(t, text, "/triangle/view")
}

func TestDriverPagesShowState(t *testing.T) {
	srv := httptest.NewServer(newPage(t.Context()))
	defer srv.Close()
	client := srv.Client()
	client.Timeout = 20 * time.Second
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	report := driver.Doctor(t.Context())
	for _, iface := range report {
		slug := driverSlug(iface.Name)
		_, known := driverSpecs[slug]
		require.Truef(t, known, "missing panel %s (%s)", slug, iface.Name)
		res, err := client.Get(srv.URL + "/driver/" + slug)
		require.NoError(t, err)
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		require.Equal(t, http.StatusOK, res.StatusCode, slug)
		text := string(body)
		require.Contains(t, text, ">Refresh<", slug)
		require.NotContains(t, text, "Read status", slug)
		require.NotContains(t, text, "No probe", slug)
		require.NotContains(t, text, "no panel", slug)
	}

	battery, err := client.Get(srv.URL + "/driver/battery")
	require.NoError(t, err)
	batteryBody, err := io.ReadAll(battery.Body)
	require.NoError(t, err)
	require.NoError(t, battery.Body.Close())
	batteryText := string(batteryBody)
	require.NotContains(t, batteryText, "<form")
	hasRows := strings.Contains(batteryText, ">Status<") && strings.Contains(batteryText, ">Level<")
	require.True(t, hasRows || strings.Contains(batteryText, `role="alert"`))

	volume, err := client.Get(srv.URL + "/driver/volume")
	require.NoError(t, err)
	volumeBody, err := io.ReadAll(volume.Body)
	require.NoError(t, err)
	require.NoError(t, volume.Body.Close())
	require.Contains(t, string(volumeBody), `action="/driver/volume/set"`)
	require.Contains(t, string(volumeBody), ">Refresh<")

	screen, err := client.Get(srv.URL + "/driver/screen")
	require.NoError(t, err)
	screenBody, err := io.ReadAll(screen.Body)
	require.NoError(t, err)
	require.NoError(t, screen.Body.Close())
	require.Contains(t, string(screenBody), "Turn display off")

	missing, err := client.Get(srv.URL + "/driver/not-a-driver")
	require.NoError(t, err)
	require.NoError(t, missing.Body.Close())
	require.Equal(t, http.StatusNotFound, missing.StatusCode)

	posted, err := client.PostForm(srv.URL+"/driver/fetchurl/fetch", url.Values{})
	require.NoError(t, err)
	require.NoError(t, posted.Body.Close())
	require.Equal(t, http.StatusSeeOther, posted.StatusCode)
	require.Contains(t, posted.Header.Get("Location"), "no+URLs+provided")

	unknown, err := client.Post(srv.URL+"/driver/battery/nope", "application/x-www-form-urlencoded", nil)
	require.NoError(t, err)
	require.NoError(t, unknown.Body.Close())
	require.Equal(t, http.StatusNotFound, unknown.StatusCode)
}

func TestBatteryPageShowsLevel(t *testing.T) {
	t.Setenv("LEWKIT_FORCE_BATTERY_DRIVER", "battery_fixed")
	srv := httptest.NewServer(newPage(t.Context()))
	defer srv.Close()
	client := srv.Client()
	client.Timeout = 5 * time.Second
	res, err := client.Get(srv.URL + "/driver/battery")
	require.NoError(t, err)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	text := string(body)
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Contains(t, text, ">Status<")
	require.Contains(t, text, ">Discharging<")
	require.Contains(t, text, ">Level<")
	require.Contains(t, text, ">42%<")
	require.NotContains(t, text, "<form")
}

func TestDriverSlugsUnique(t *testing.T) {
	seen := map[string]string{}
	for _, iface := range driver.Doctor(t.Context()) {
		slug := driverSlug(iface.Name)
		prev, ok := seen[slug]
		require.Falsef(t, ok, "slug %s for %s and %s", slug, prev, iface.Name)
		seen[slug] = iface.Name
	}
	_, ok := seen["battery"]
	require.True(t, ok, "battery missing from doctor")
}
