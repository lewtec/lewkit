package android

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/stretchr/testify/require"
)

func TestPackResponseKeepsBodyAfterHeaders(t *testing.T) {
	header := textHeader("text/html; charset=utf-8")
	header.Add("Set-Cookie", "a=1")
	body := []byte("<p>hi</p>\n\n")
	packed := packResponse(http.StatusOK, header, body)
	text := string(packed)
	head, payload, ok := strings.Cut(text, "\n\n")
	require.True(t, ok)
	require.True(t, strings.HasPrefix(head, "200\n"))
	require.Contains(t, head, "Content-Type: text/html; charset=utf-8")
	require.Contains(t, head, "Set-Cookie: a=1")
	require.Equal(t, string(body), payload)
	require.NotContains(t, head, "Content-Length")
}

func TestFetchPostsTheFormBody(t *testing.T) {
	var got string
	view := &pageView{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = r.Method + " " + r.URL.Path + " " + string(body)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<head></head><p>ok</p>")
	})}
	packed := view.fetch([]any{"POST", pageOrigin + "/driver/share/send", "Content-Type: application/x-www-form-urlencoded\n", "name=lew"})
	require.Equal(t, "POST /driver/share/send name=lew", got)
	require.Contains(t, string(packed), "<script>")
	require.Contains(t, string(packed), "lewkitPage.submit")
	require.Contains(t, string(packed), pageOrigin+"/driver/share/send")
}

func TestFetchFollowsTheRedirectTarget(t *testing.T) {
	view := &pageView{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/driver/share/send" {
			http.Redirect(w, r, "/driver/share?notice=done", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<head></head><p>"+r.URL.RawQuery+"</p>")
	})}
	packed := string(view.fetch([]any{"POST", pageOrigin + "/driver/share/send", "", "op=1"}))
	require.Contains(t, packed, pageHeader+": "+pageOrigin+"/driver/share?notice=done")
	require.Contains(t, packed, "<p>notice=done</p>")
	require.NotContains(t, packed, "\n303\n")
}

func TestBridgeScriptNamesThePageObject(t *testing.T) {
	script := bridgeScript()
	require.Contains(t, script, "window."+webview.ScriptName())
	require.Contains(t, script, "lewkitPage.submit")
	require.NotContains(t, script, "</script>")
}
