package webview

import (
	"io"
	"net/http"
	"strings"
	"sync"
)

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
