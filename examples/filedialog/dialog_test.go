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
	}, func(names ...string) (fs.FS, error) {
		return fileTree(), nil
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
	require.NotContains(t, body, "readme.txt")
	got := <-called
	require.True(t, got.Folder)
	require.Equal(t, "Choose folder", got.Title)

	unblock()
	require.Eventually(t, func() bool {
		body := httpPage.get()
		return strings.Contains(body, "readme.txt") && !strings.Contains(body, "content://tree")
	}, time.Second, 10*time.Millisecond)
}

func TestFolderListsNames(t *testing.T) {
	opened := make(chan []string, 1)
	httpPage := openFiles(t, func(context.Context, filedialog.Request) ([]string, error) {
		return []string{"content://tree"}, nil
	}, func(names ...string) (fs.FS, error) {
		opened <- append([]string(nil), names...)
		return fileTree(), nil
	})

	httpPage.post(url.Values{"op": {"folder"}})
	require.Eventually(t, func() bool {
		return strings.Contains(httpPage.get(), "readme.txt")
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, []string{"content://tree"}, <-opened)

	body := httpPage.get()
	require.Contains(t, body, "pics")
	require.Contains(t, body, "hello")
	require.Contains(t, body, "5 B")
	require.NotContains(t, body, "a.jpg")
	require.NotContains(t, body, "content://")

	body = httpPage.textAt("/?path=pics")
	require.Contains(t, body, "a.jpg")
	require.Contains(t, body, "<img")
	require.Contains(t, body, "/file?path="+url.QueryEscape("pics/a.jpg"))
	require.Contains(t, body, "Up")
	require.Equal(t, "hello", httpPage.textAt("/file?path=readme.txt"))
	require.Equal(t, "jpg", httpPage.textAt("/file?path="+url.QueryEscape("pics/a.jpg")))

	miss := httpPage.textAt("/?path=..")
	require.Contains(t, miss, "open ..")
	require.NotContains(t, miss, "readme.txt")
	require.Contains(t, miss, "Open files")
}

func TestTextShowsEscaped(t *testing.T) {
	httpPage := openFiles(t, func(context.Context, filedialog.Request) ([]string, error) {
		return []string{"content://note"}, nil
	}, func(names ...string) (fs.FS, error) {
		return fstest.MapFS{"note.txt": &fstest.MapFile{Data: []byte("<b>hi</b>")}}, nil
	})

	httpPage.post(url.Values{"op": {"files"}})
	require.Eventually(t, func() bool {
		return strings.Contains(httpPage.get(), "note.txt")
	}, time.Second, 10*time.Millisecond)
	body := httpPage.get()
	require.Contains(t, body, "&lt;b&gt;hi&lt;/b&gt;")
	require.NotContains(t, body, "<b>")
	require.Contains(t, body, "Open files")
}

func TestSaveSendsTheName(t *testing.T) {
	called := make(chan filedialog.Request, 1)
	httpPage := openFiles(t, func(_ context.Context, req filedialog.Request) ([]string, error) {
		called <- req
		return []string{"content://notes"}, nil
	}, func(names ...string) (fs.FS, error) {
		return fstest.MapFS{"notes.txt": &fstest.MapFile{Data: []byte("saved")}}, nil
	})

	httpPage.post(url.Values{"op": {"save"}, "name": {"notes.txt"}})
	got := <-called
	require.True(t, got.Save)
	require.Equal(t, "notes.txt", got.Name)
	require.False(t, got.Folder)
	require.Eventually(t, func() bool {
		return strings.Contains(httpPage.get(), "notes.txt")
	}, time.Second, 10*time.Millisecond)
}

func TestOpenFilesAllowsMoreThanOne(t *testing.T) {
	called := make(chan filedialog.Request, 1)
	opened := make(chan []string, 1)
	httpPage := openFiles(t, func(_ context.Context, req filedialog.Request) ([]string, error) {
		called <- req
		return []string{"content://a", "content://b"}, nil
	}, func(names ...string) (fs.FS, error) {
		opened <- append([]string(nil), names...)
		return fstest.MapFS{
			"a.txt": &fstest.MapFile{Data: []byte("a")},
			"b.txt": &fstest.MapFile{Data: []byte("b")},
		}, nil
	})

	httpPage.post(url.Values{"op": {"files"}})
	require.Eventually(t, func() bool {
		body := httpPage.get()
		return strings.Contains(body, "a.txt") && strings.Contains(body, "b.txt") && !strings.Contains(body, "content://")
	}, time.Second, 10*time.Millisecond)
	got := <-called
	require.True(t, got.Multiple)
	require.False(t, got.Save)
	require.Equal(t, []string{"content://a", "content://b"}, <-opened)
}

func TestPickErrorIsShown(t *testing.T) {
	httpPage := openFiles(t, func(context.Context, filedialog.Request) ([]string, error) {
		return nil, errNoPicker
	}, func(names ...string) (fs.FS, error) {
		return nil, errOpened
	})

	httpPage.post(url.Values{"op": {"folder"}})
	require.Eventually(t, func() bool {
		body := httpPage.get()
		return strings.Contains(body, errNoPicker.Error()) && !strings.Contains(body, errOpened.Error())
	}, time.Second, 10*time.Millisecond)
}

func TestOpenErrorIsShown(t *testing.T) {
	httpPage := openFiles(t, func(context.Context, filedialog.Request) ([]string, error) {
		return []string{"content://tree"}, nil
	}, func(names ...string) (fs.FS, error) {
		return nil, errOpened
	})

	httpPage.post(url.Values{"op": {"folder"}})
	require.Eventually(t, func() bool {
		return strings.Contains(httpPage.get(), errOpened.Error())
	}, time.Second, 10*time.Millisecond)
}

func TestUnknownOpDoesNotOpen(t *testing.T) {
	httpPage := openFiles(t, func(context.Context, filedialog.Request) ([]string, error) {
		return nil, errOpened
	}, nil)

	body := httpPage.post(url.Values{"op": {"nope"}})
	require.NotContains(t, body, errOpened.Error())
	require.NotContains(t, body, "Waiting for the picker")
}

func fileTree() fs.FS {
	return fstest.MapFS{
		"readme.txt": &fstest.MapFile{Data: []byte("hello")},
		"pics/a.jpg": &fstest.MapFile{Data: []byte("jpg")},
	}
}

type filesHTTP struct {
	t      *testing.T
	client *http.Client
	base   string
}

func openFiles(t *testing.T, choose chooseFunc, open openFunc) *filesHTTP {
	t.Helper()
	srv := httptest.NewServer(newFiles(t.Context(), choose, open))
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
