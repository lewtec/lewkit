package webview

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

const maxRedirects = 8

// Dispatch calls handler as the web view would, without listening on a port.
// An app:// request is presented to the handler as http://127.0.0.1 plus the
// path and query. A same-view redirect is followed here. A redirect to another
// origin is returned with its Location rewritten onto the view origin when it
// is still this view, and left absolute when it is not.
func Dispatch(handler http.Handler, method, target string, header http.Header, body io.Reader) (int, http.Header, io.ReadCloser, error) {
	if handler == nil {
		return http.StatusNotFound, nil, http.NoBody, ErrPage
	}
	if body == nil {
		body = http.NoBody
	}
	payload, err := io.ReadAll(body)
	if closer, ok := body.(io.Closer); ok {
		closer.Close()
	}
	if err != nil {
		return http.StatusBadRequest, nil, http.NoBody, err
	}
	return dispatch(handler, method, target, header, payload, target, 0)
}

func dispatch(handler http.Handler, method, target string, header http.Header, payload []byte, origin string, hop int) (int, http.Header, io.ReadCloser, error) {
	if method == "" {
		method = http.MethodGet
	}
	request, err := http.NewRequest(method, normalizeTarget(target), bytes.NewReader(payload))
	if err != nil {
		return http.StatusBadRequest, nil, http.NoBody, err
	}
	if header != nil {
		request.Header = header.Clone()
	}
	if requestHasBody(method) && len(payload) > 0 && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	reader, writer := io.Pipe()
	stream := &streamResponse{
		header: make(http.Header),
		status: http.StatusOK,
		ready:  make(chan struct{}),
		body:   writer,
	}
	go func() {
		defer writer.Close()
		handler.ServeHTTP(stream, request)
		stream.finishHeaders()
	}()
	<-stream.ready
	status, sent := stream.status, stream.sent
	if hop < maxRedirects && isRedirect(status) {
		next, ok := sameViewTarget(origin, request, sent.Get("Location"))
		if ok {
			_, _ = io.Copy(io.Discard, reader)
			_ = reader.Close()
			nextMethod, nextPayload := method, payload
			if !keepsMethod(status) {
				nextMethod = http.MethodGet
				nextPayload = nil
			}
			return dispatch(handler, nextMethod, next, nil, nextPayload, origin, hop+1)
		}
	}
	if sent != nil {
		if loc := sent.Get("Location"); loc != "" {
			sent.Set("Location", rewriteLocation(origin, loc))
		}
	}
	return status, sent, reader, nil
}

func normalizeTarget(target string) string {
	parsed, err := url.Parse(target)
	if err != nil {
		return "http://127.0.0.1/"
	}
	path := parsed.Path
	if path == "" {
		path = "/"
	}
	out := url.URL{Scheme: "http", Host: "127.0.0.1", Path: path, RawQuery: parsed.RawQuery}
	return out.String()
}

func requestHasBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func isRedirect(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func keepsMethod(status int) bool {
	return status == http.StatusTemporaryRedirect || status == http.StatusPermanentRedirect
}

// sameViewTarget returns the next in-process URL when loc stays on this view.
func sameViewTarget(origin string, current *http.Request, loc string) (string, bool) {
	parsed, err := url.Parse(loc)
	if err != nil {
		return "", false
	}
	if parsed.IsAbs() && !sameLoopback(parsed) {
		base, err := url.Parse(origin)
		if err != nil || base.Scheme != parsed.Scheme || base.Host != parsed.Host {
			return "", false
		}
	}
	if !parsed.IsAbs() && (parsed.Path == "" || parsed.Path[0] != '/') && current != nil && current.URL != nil {
		parsed = current.URL.ResolveReference(parsed)
	}
	path := parsed.Path
	if path == "" {
		path = "/"
	}
	next := url.URL{Scheme: "http", Host: "127.0.0.1", Path: path, RawQuery: parsed.RawQuery, Fragment: parsed.Fragment}
	return next.String(), true
}

func sameLoopback(u *url.URL) bool {
	host := u.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return false
	}
	return u.Port() == "" || u.Port() == "80"
}

func rewriteLocation(origin, loc string) string {
	parsed, err := url.Parse(loc)
	if err != nil {
		return loc
	}
	base, err := url.Parse(origin)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return loc
	}
	if parsed.IsAbs() && !sameLoopback(parsed) && (parsed.Scheme != base.Scheme || parsed.Host != base.Host) {
		return loc
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	parsed.Scheme = base.Scheme
	parsed.Host = base.Host
	return parsed.String()
}
