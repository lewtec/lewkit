package webview

import (
	"io"
	"net/http"
	"strings"
	"sync"
)

// Dispatch calls handler as the web view would, without listening on a port.
// target is the request URL. header and body may be empty.
// The returned body is the handler output as it is written. Close it
// when the web view is done reading.
func Dispatch(handler http.Handler, method, target string, header http.Header, body io.Reader) (int, http.Header, io.ReadCloser, error) {
	if handler == nil {
		return http.StatusNotFound, nil, http.NoBody, ErrPage
	}
	if method == "" {
		method = http.MethodGet
	}
	if body == nil {
		body = http.NoBody
	}
	request, err := http.NewRequest(method, target, body)
	if err != nil {
		return http.StatusBadRequest, nil, http.NoBody, err
	}
	if header != nil {
		request.Header = header.Clone()
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
		if closer, ok := body.(io.Closer); ok {
			defer closer.Close()
		}
		handler.ServeHTTP(stream, request)
		stream.finishHeaders()
	}()
	<-stream.ready
	return stream.status, stream.sent, reader, nil
}

// streamResponse passes handler writes through to a pipe.
// Headers are published on the first WriteHeader or Write.
type streamResponse struct {
	header http.Header
	status int
	sent   http.Header
	ready  chan struct{}
	once   sync.Once
	body   *io.PipeWriter
}

func (response *streamResponse) Header() http.Header { return response.header }

func (response *streamResponse) WriteHeader(status int) {
	response.finishHeadersLocked(status)
}

func (response *streamResponse) Write(payload []byte) (int, error) {
	response.finishHeadersLocked(http.StatusOK)
	return response.body.Write(payload)
}

func (response *streamResponse) finishHeaders() {
	response.finishHeadersLocked(http.StatusOK)
}

func (response *streamResponse) finishHeadersLocked(status int) {
	response.once.Do(func() {
		response.status = status
		response.sent = response.header.Clone()
		close(response.ready)
	})
}

// HeaderText renders header as HTTP header lines for a web view response.
func HeaderText(header http.Header) string {
	if len(header) == 0 {
		return ""
	}
	var lines strings.Builder
	for name, values := range header {
		for _, value := range values {
			lines.WriteString(name)
			lines.WriteString(": ")
			lines.WriteString(value)
			lines.WriteString("\r\n")
		}
	}
	return lines.String()
}
