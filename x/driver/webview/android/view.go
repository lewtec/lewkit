package android

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/ffi/jni"
)

type pageView struct {
	id       int
	handler  http.Handler
	html     string
	files    fs.FS
	messages chan []byte
	done     chan struct{}
	once     sync.Once
	proxy    *jni.Ref
	evals    sync.Map
}

var (
	pageSeq atomic.Int64
	evalSeq atomic.Uint64
)

func (v *pageView) Messages() <-chan []byte { return v.messages }

func (v *pageView) Done() <-chan struct{} { return v.done }

func (v *pageView) Close() error {
	select {
	case <-v.done:
		return nil
	default:
	}
	_, err := jni.CallStatic("lewkit.Page", "close", v.id)
	if err != nil {
		v.markClosed()
	}
	return err
}

func (v *pageView) Evaluate(ctx context.Context, script string) (string, error) {
	select {
	case <-v.done:
		return "", webview.ErrClosed
	default:
	}
	token := strconv.FormatUint(evalSeq.Add(1), 10)
	ch := make(chan string, 1)
	v.evals.Store(token, ch)
	defer v.evals.Delete(token)
	if _, err := jni.CallStatic("lewkit.Page", "evaluate", v.id, token, script); err != nil {
		return "", err
	}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-v.done:
		return "", webview.ErrClosed
	case text := <-ch:
		return text, nil
	}
}

func (v *pageView) invoke(method string, args []any) (any, error) {
	switch method {
	case "fetch":
		return v.fetch(args), nil
	case "message":
		v.deliver(argText(args, 0))
	case "closed":
		v.markClosed()
	case "evalResult":
		v.finishEval(argText(args, 0), argText(args, 1))
	}
	return nil, nil
}

func (v *pageView) markClosed() {
	v.once.Do(func() {
		close(v.done)
		proxy := v.proxy
		v.proxy = nil
		if proxy != nil {
			proxy.Release()
		}
	})
}

func (v *pageView) deliver(text string) {
	body := []byte(text)
	select {
	case v.messages <- body:
	default:
		go func() {
			select {
			case v.messages <- body:
			case <-v.done:
			}
		}()
	}
}

func (v *pageView) finishEval(token, text string) {
	loaded, ok := v.evals.LoadAndDelete(token)
	if !ok {
		return
	}
	ch, _ := loaded.(chan string)
	if ch == nil {
		return
	}
	select {
	case ch <- text:
	default:
	}
}

func (v *pageView) fetch(args []any) []byte {
	method := argText(args, 0)
	rawURL := argText(args, 1)
	if method == "" {
		method = http.MethodGet
	}
	if v.handler != nil {
		return v.fetchHandler(method, rawURL, argText(args, 2), argText(args, 3))
	}
	return v.fetchFiles(rawURL)
}

func (v *pageView) fetchHandler(method, rawURL, headerText, body string) []byte {
	landed := ""
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL != nil {
			landed = r.URL.Path
			if r.URL.RawQuery != "" {
				landed += "?" + r.URL.RawQuery
			}
		}
		v.handler.ServeHTTP(w, r)
	})
	header := parseHeaderText(headerText)
	status, sent, reader, err := webview.Dispatch(wrapped, method, rawURL, header, strings.NewReader(body))
	if err != nil {
		return packResponse(http.StatusBadGateway, textHeader("text/plain; charset=utf-8"), []byte(err.Error()))
	}
	defer reader.Close()
	payload, err := io.ReadAll(reader)
	if err != nil {
		return packResponse(http.StatusBadGateway, textHeader("text/plain; charset=utf-8"), []byte(err.Error()))
	}
	if sent == nil {
		sent = make(http.Header)
	}
	if landed != "" {
		sent.Set(pageHeader, pageURL(landed))
	}
	return packResponse(status, sent, injectBridge(sent, payload))
}

func (v *pageView) fetchFiles(rawURL string) []byte {
	path := "/"
	if parsed, err := url.Parse(rawURL); err == nil && parsed.Path != "" {
		path = parsed.Path
	}
	payload, kind, err := webview.ReadPage(v.html, v.files, path)
	if err != nil {
		return packResponse(http.StatusNotFound, textHeader("text/plain; charset=utf-8"), []byte(err.Error()))
	}
	header := textHeader(kind)
	header.Set(pageHeader, pageURL(path))
	return packResponse(http.StatusOK, header, injectBridge(header, payload))
}

func pageURL(path string) string {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return pageOrigin + path
}

func textHeader(kind string) http.Header {
	header := make(http.Header)
	if kind != "" {
		header.Set("Content-Type", kind)
	}
	return header
}

func argText(args []any, i int) string {
	if i < 0 || i >= len(args) || args[i] == nil {
		return ""
	}
	text, _ := args[i].(string)
	return text
}
