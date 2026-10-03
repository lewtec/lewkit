package main

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/http/asset/daisyui"
	"github.com/lewtec/lewkit/x/http/asset/tailwindcss"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/require"
)

var (
	errPicker = errors.New("picker")
	errOpened = errors.New("opened")
)

func TestWelcomePageStartsBeforeThePicker(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	called := make(chan filedialog.Request, 1)
	page := openWelcome(t, welcomeConfig{
		choose: func(_ context.Context, req filedialog.Request) ([]string, error) {
			called <- req
			<-release
			return []string{"content://tree"}, nil
		},
		open: func(...string) (fs.FS, error) {
			return fstest.MapFS{"notes.txt": &fstest.MapFile{Data: []byte("hi")}}, nil
		},
	})

	body := page.get()
	require.Contains(t, body, "Open a folder")
	require.Contains(t, body, daisyui.Path)
	require.Contains(t, body, tailwindcss.Path)
	require.NotContains(t, body, "cdn.jsdelivr.net")
	require.NotContains(t, body, "Waiting for the picker")
	select {
	case req := <-called:
		require.Failf(t, "choose called", "%+v", req)
	default:
	}

	body = page.post()
	require.Contains(t, body, "Waiting for the picker")
	require.NotContains(t, body, "content://tree")
	require.NotContains(t, body, "notes.txt")
	got := <-called
	require.True(t, got.Folder)
	require.Equal(t, "Open folder", got.Title)

	unblock()
	require.Eventually(t, func() bool {
		body := page.get()
		return strings.Contains(body, "notes.txt") && strings.Contains(body, "content://tree")
	}, time.Second, 10*time.Millisecond)
}

func TestWelcomeRecentOpensWithoutTheDialog(t *testing.T) {
	called := false
	page := openWelcome(t, welcomeConfig{
		recent: []gui.Directory{{Path: "/projects/demo"}},
		choose: func(context.Context, filedialog.Request) ([]string, error) {
			called = true
			return nil, errPicker
		},
		open: func(names ...string) (fs.FS, error) {
			return fstest.MapFS{"ledger": &fstest.MapFile{Mode: fs.ModeDir}}, nil
		},
	})

	body := page.textAt("/?path=" + url.QueryEscape("/projects/demo"))
	require.False(t, called)
	require.Contains(t, body, "ledger")
	require.Contains(t, body, "/projects/demo")
	require.Contains(t, body, "Open a folder")
}

func TestWelcomeUnavailableNote(t *testing.T) {
	page := openWelcome(t, welcomeConfig{
		choose: func(context.Context, filedialog.Request) ([]string, error) {
			return nil, driver.ErrUnavailable
		},
		open: func(...string) (fs.FS, error) {
			return nil, errOpened
		},
	})
	page.post()
	require.Eventually(t, func() bool {
		body := page.get()
		return strings.Contains(body, "No folder dialog on this system.") && !strings.Contains(body, "opened")
	}, time.Second, 10*time.Millisecond)
}

func openWelcome(t *testing.T, cfg welcomeConfig) *welcomeHTTP {
	t.Helper()
	if cfg.title == "" {
		cfg.title = "Welcome"
	}
	srv := httptest.NewServer(newWelcome(t.Context(), cfg))
	t.Cleanup(srv.Close)
	return &welcomeHTTP{
		t:      t,
		client: &http.Client{Timeout: 5 * time.Second},
		base:   srv.URL,
	}
}

type welcomeHTTP struct {
	t      *testing.T
	client *http.Client
	base   string
}

func (h *welcomeHTTP) get() string {
	h.t.Helper()
	return h.textAt("/")
}

func (h *welcomeHTTP) textAt(path string) string {
	h.t.Helper()
	res, err := h.client.Get(h.base + path)
	require.NoError(h.t, err)
	return h.text(res)
}

func (h *welcomeHTTP) post() string {
	h.t.Helper()
	res, err := h.client.PostForm(h.base+"/", nil)
	require.NoError(h.t, err)
	return h.text(res)
}

func (h *welcomeHTTP) text(res *http.Response) string {
	h.t.Helper()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(h.t, err)
	return string(body)
}
