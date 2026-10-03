package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/http/asset/daisyui"
	"github.com/lewtec/lewkit/x/http/asset/tailwindcss"
	"github.com/stretchr/testify/require"
)

var (
	errNoPicker = errors.New("no picker")
	errOpened   = errors.New("opened")
)

func TestPageStartsBeforeThePicker(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	called := make(chan filedialog.Request, 1)
	httpPage := openFiles(t, func(_ context.Context, req filedialog.Request) ([]string, error) {
		called <- req
		<-release
		return []string{"content://tree"}, nil
	})

	body := httpPage.get()
	require.Contains(t, body, "Open files")
	require.Contains(t, body, "Choose folder")
	require.Contains(t, body, "Save")
	require.Contains(t, body, daisyui.Path)
	require.Contains(t, body, tailwindcss.Path)
	require.NotContains(t, body, "cdn.jsdelivr.net")
	require.Contains(t, httpPage.textAt(daisyui.Path), daisyui.Version)
	require.Contains(t, httpPage.textAt(tailwindcss.Path), tailwindcss.Version)
	require.NotContains(t, body, "Waiting for the picker")
	select {
	case req := <-called:
		require.Failf(t, "choose called", "%+v", req)
	default:
	}

	body = httpPage.post(url.Values{"op": {"folder"}})
	require.Contains(t, body, "Waiting for the picker")
	require.NotContains(t, body, "content://tree")
	got := <-called
	require.True(t, got.Folder)
	require.Equal(t, "Choose folder", got.Title)

	unblock()
	require.Eventually(t, func() bool {
		return strings.Contains(httpPage.get(), "content://tree")
	}, time.Second, 10*time.Millisecond)
}

func TestSaveSendsTheName(t *testing.T) {
	called := make(chan filedialog.Request, 1)
	httpPage := openFiles(t, func(_ context.Context, req filedialog.Request) ([]string, error) {
		called <- req
		return []string{"content://notes"}, nil
	})

	httpPage.post(url.Values{"op": {"save"}, "name": {"notes.txt"}})
	got := <-called
	require.True(t, got.Save)
	require.Equal(t, "notes.txt", got.Name)
	require.False(t, got.Folder)
}

func TestOpenFilesAllowsMoreThanOne(t *testing.T) {
	called := make(chan filedialog.Request, 1)
	httpPage := openFiles(t, func(_ context.Context, req filedialog.Request) ([]string, error) {
		called <- req
		return []string{"content://a", "content://b"}, nil
	})

	httpPage.post(url.Values{"op": {"files"}})
	require.Eventually(t, func() bool {
		body := httpPage.get()
		return strings.Contains(body, "content://a") && strings.Contains(body, "content://b")
	}, time.Second, 10*time.Millisecond)
	got := <-called
	require.True(t, got.Multiple)
	require.False(t, got.Save)
}

func TestPickErrorIsShown(t *testing.T) {
	httpPage := openFiles(t, func(context.Context, filedialog.Request) ([]string, error) {
		return nil, errNoPicker
	})

	httpPage.post(url.Values{"op": {"folder"}})
	require.Eventually(t, func() bool {
		return strings.Contains(httpPage.get(), errNoPicker.Error())
	}, time.Second, 10*time.Millisecond)
}

func TestUnknownOpDoesNotOpen(t *testing.T) {
	httpPage := openFiles(t, func(context.Context, filedialog.Request) ([]string, error) {
		return nil, errOpened
	})

	body := httpPage.post(url.Values{"op": {"nope"}})
	require.NotContains(t, body, errOpened.Error())
	require.NotContains(t, body, "Waiting for the picker")
}

type filesHTTP struct {
	t      *testing.T
	client *http.Client
	base   string
}

func openFiles(t *testing.T, choose chooseFunc) *filesHTTP {
	t.Helper()
	srv := httptest.NewServer(newFiles(t.Context(), choose))
	t.Cleanup(srv.Close)
	return &filesHTTP{
		t:      t,
		client: &http.Client{Timeout: 5 * time.Second},
		base:   srv.URL,
	}
}

func (h *filesHTTP) get() string {
	h.t.Helper()
	return h.textAt("/")
}

func (h *filesHTTP) textAt(path string) string {
	h.t.Helper()
	res, err := h.client.Get(h.base + path)
	require.NoError(h.t, err)
	return h.text(res)
}

func (h *filesHTTP) post(form url.Values) string {
	h.t.Helper()
	res, err := h.client.PostForm(h.base+"/", form)
	require.NoError(h.t, err)
	return h.text(res)
}

func (h *filesHTTP) text(res *http.Response) string {
	h.t.Helper()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(h.t, err)
	return string(body)
}
