package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInboxNameFromURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"eletrocromo-inbox://inbox?name=photo.jpg", "photo.jpg"},
		{"eletrocromo-inbox://inbox?file=a.png", "a.png"},
		{"eletrocromo-inbox://inbox?path=../x.txt", "x.txt"},
		{"eletrocromo-inbox://inbox/foo.md", "foo.md"},
		{"eletrocromo-inbox:///inbox/foo.md", "foo.md"},
		{"eletrocromo-inbox:inbox/bar.txt", "bar.txt"},
		{"eletrocromo-inbox://from-webview", ""},
		{"eletrocromo-inbox://x/y?q=1", ""},
		{"://no-scheme", ""},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, inboxNameFromURL(tt.in))
		})
	}
}

func TestOpenRecordsURL(t *testing.T) {
	st := &state{}
	srv := httptest.NewServer(newInbox(st))
	defer srv.Close()
	client := srv.Client()
	client.Timeout = 5 * time.Second
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	res, err := client.PostForm(srv.URL+"/open", url.Values{"url": {"eletrocromo-inbox://from-webview"}})
	require.NoError(t, err)
	res.Body.Close()
	require.Equal(t, http.StatusSeeOther, res.StatusCode)
	got := st.list()
	require.Len(t, got, 1)
	require.Equal(t, "url", got[0].Kind)
	require.Equal(t, "eletrocromo-inbox://from-webview", got[0].Text)
}
