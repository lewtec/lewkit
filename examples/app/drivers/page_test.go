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
	for _, action := range formActions(batteryText) {
		require.Equal(t, "/driver/battery/pin", action)
	}
	hasRows := strings.Contains(batteryText, ">Status<") && strings.Contains(batteryText, ">Level<")
	require.True(t, hasRows || strings.Contains(batteryText, `role="alert"`))

	volume, err := client.Get(srv.URL + "/driver/volume")
	require.NoError(t, err)
	volumeBody, err := io.ReadAll(volume.Body)
	require.NoError(t, err)
	require.NoError(t, volume.Body.Close())
	require.Contains(t, string(volumeBody), `action="/driver/volume/set"`)
	require.Contains(t, string(volumeBody), ">Refresh<")

	box, err := client.Get(srv.URL + "/driver/messagebox")
	require.NoError(t, err)
	boxBody, err := io.ReadAll(box.Body)
	require.NoError(t, err)
	require.NoError(t, box.Body.Close())
	require.Contains(t, string(boxBody), `action="/driver/messagebox/show"`)
	require.Contains(t, string(boxBody), `<select class="select w-full" name="style">`)
	require.Contains(t, string(boxBody), `<option value="informational" selected>informational</option>`)
	require.Contains(t, string(boxBody), `<option value="warning">warning</option>`)
	require.Contains(t, string(boxBody), `<option value="critical">critical</option>`)

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
	for _, action := range formActions(text) {
		require.Equal(t, "/driver/battery/pin", action)
	}
	require.NotContains(t, text, `value="battery_fixed"`)
}

func TestUseButtonPinsDriver(t *testing.T) {
	batteryName := interfaceNameBySlug(t, "battery")
	presentName := interfaceNameBySlug(t, "present")
	evalName := interfaceNameBySlug(t, "ndarray.Evaluator")
	t.Cleanup(func() {
		driver.Pin(batteryName, "")
		driver.Pin(presentName, "")
		driver.Pin(evalName, "")
	})

	srv := httptest.NewServer(newPage(t.Context()))
	defer srv.Close()
	client := srv.Client()
	client.Timeout = 20 * time.Second
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	posted, err := client.PostForm(srv.URL+"/driver/battery/pin", url.Values{"id": {"battery_fixed"}})
	require.NoError(t, err)
	require.Equal(t, http.StatusSeeOther, posted.StatusCode)
	require.Contains(t, posted.Header.Get("Location"), "notice=using+battery_fixed")
	require.NoError(t, posted.Body.Close())

	res, err := client.Get(srv.URL + "/driver/battery?notice=using+battery_fixed")
	require.NoError(t, err)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	text := string(body)
	require.Contains(t, text, "using battery_fixed")
	require.Contains(t, text, "battery_fixed · 101")
	require.Contains(t, text, "badge-success")
	require.NotContains(t, text, `value="battery_fixed"`)

	posted, err = client.PostForm(srv.URL+"/driver/present/pin", url.Values{"id": {"present_opengl"}})
	require.NoError(t, err)
	require.Equal(t, http.StatusSeeOther, posted.StatusCode)
	require.Contains(t, posted.Header.Get("Location"), "using+present_opengl")
	require.NoError(t, posted.Body.Close())

	res, err = client.Get(srv.URL + posted.Header.Get("Location"))
	require.NoError(t, err)
	body, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Contains(t, string(body), "present_opengl · 101")

	missing, err := client.PostForm(srv.URL+"/driver/present/nope", url.Values{})
	require.NoError(t, err)
	require.NoError(t, missing.Body.Close())
	require.Equal(t, http.StatusNotFound, missing.StatusCode)

	posted, err = client.PostForm(srv.URL+"/driver/ndarray.Evaluator/pin", url.Values{"id": {"ndeval_opengl"}})
	require.NoError(t, err)
	require.Equal(t, http.StatusSeeOther, posted.StatusCode)
	require.Contains(t, posted.Header.Get("Location"), "using+ndeval_opengl")
	require.NoError(t, posted.Body.Close())
}

func interfaceNameBySlug(t *testing.T, slug string) string {
	t.Helper()
	for _, iface := range driver.Doctor(t.Context()) {
		if driverSlug(iface.Name) == slug {
			return iface.Name
		}
	}
	t.Fatalf("slug %s missing", slug)
	return ""
}

func formActions(html string) []string {
	var out []string
	rest := html
	for {
		i := strings.Index(rest, "<form ")
		if i < 0 {
			return out
		}
		rest = rest[i:]
		end := strings.Index(rest, ">")
		if end < 0 {
			return out
		}
		tag := rest[:end]
		rest = rest[end+1:]
		const key = `action="`
		j := strings.Index(tag, key)
		if j < 0 {
			continue
		}
		tag = tag[j+len(key):]
		k := strings.IndexByte(tag, '"')
		if k < 0 {
			continue
		}
		out = append(out, tag[:k])
	}
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
