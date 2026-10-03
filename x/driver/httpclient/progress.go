package httpclient

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/taskgroup"
)

// progressTransport promotes a request to an Internet task when the request
// context carries a taskgroup session. Body reads drive the progress bar.
type progressTransport struct {
	base http.RoundTripper
}

func (transport *progressTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if taskgroup.FromContext(request.Context()) == nil {
		return transport.base.RoundTrip(request)
	}

	name := taskName(request)
	resultChannel := make(chan roundTripResult, 1)
	bodyComplete := make(chan struct{})

	taskgroup.Go(request.Context(), name, taskgroup.Internet, func(ctx context.Context, status *taskgroup.Status) error {
		slog.DebugContext(ctx, "http request", "url", request.URL.String())
		status.Update("connecting")
		outgoing := request.WithContext(ctx)
		response, err := transport.base.RoundTrip(outgoing)
		if err != nil {
			resultChannel <- roundTripResult{err: err}
			return err
		}
		total := response.ContentLength
		if total > 0 {
			status.Progress(0, total)
			status.Update(byteProgress(0, total))
		} else {
			status.Progress(0, 1)
			status.Update("0 B")
		}
		response.Body = &progressReadCloser{
			ReadCloser:   response.Body,
			status:       status,
			total:        total,
			completionCh: bodyComplete,
		}
		resultChannel <- roundTripResult{response: response}
		select {
		case <-bodyComplete:
		case <-ctx.Done():
		}
		if total > 0 {
			status.Progress(total, total)
			status.Update(humanBytes(total))
		} else {
			status.Progress(1, 1)
		}
		return nil
	})

	select {
	case result := <-resultChannel:
		return result.response, result.err
	case <-request.Context().Done():
		return nil, context.Cause(request.Context())
	}
}

type roundTripResult struct {
	response *http.Response
	err      error
}

type progressReadCloser struct {
	io.ReadCloser
	status       *taskgroup.Status
	total        int64
	written      int64
	completionCh chan struct{}
	once         sync.Once
}

func (reader *progressReadCloser) Read(buffer []byte) (int, error) {
	count, err := reader.ReadCloser.Read(buffer)
	if count > 0 {
		reader.written += int64(count)
		current := reader.written
		if reader.total > 0 && current > reader.total {
			current = reader.total
		}
		if reader.total > 0 {
			reader.status.Progress(current, reader.total)
			reader.status.Update(byteProgress(reader.written, reader.total))
		} else {
			reader.status.Update(humanBytes(reader.written))
		}
	}
	if err != nil {
		reader.signalComplete()
	}
	return count, err
}

func (reader *progressReadCloser) Close() error {
	err := reader.ReadCloser.Close()
	reader.signalComplete()
	return err
}

func (reader *progressReadCloser) signalComplete() {
	reader.once.Do(func() {
		if reader.completionCh != nil {
			close(reader.completionCh)
		}
	})
}

func byteProgress(written, total int64) string {
	return humanBytes(written) + " / " + humanBytes(total)
}

func humanBytes(size int64) string {
	if size <= 0 {
		return "0 B"
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	divisor, exponent := int64(unit), 0
	for next := size / unit; next >= unit; next /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(divisor), "KMGTPE"[exponent])
}

type taskLabelKey struct{}

// WithTaskLabel sets the Internet task title for HTTP requests made with ctx.
func WithTaskLabel(ctx context.Context, label string) context.Context {
	if ctx == nil || strings.TrimSpace(label) == "" {
		return ctx
	}
	return context.WithValue(ctx, taskLabelKey{}, strings.TrimSpace(label))
}

func taskName(request *http.Request) string {
	if request != nil {
		if label, ok := request.Context().Value(taskLabelKey{}).(string); ok && label != "" {
			return label
		}
	}
	if request == nil || request.URL == nil {
		return "http"
	}
	path := request.URL.Path
	host := request.URL.Host
	if strings.Contains(host, "github") && strings.Contains(path, "/releases/assets/") {
		return "github release"
	}
	if strings.Contains(host, "githubusercontent.com") {
		return "github"
	}
	if base := filepath.Base(path); base != "" && base != "." && base != "/" {
		if strings.Contains(base, ".") {
			return base
		}
		if looksLikeUUID(base) || isAllDigits(base) {
			if host != "" {
				return host
			}
			return "http"
		}
		return base
	}
	if host != "" {
		return host
	}
	return "http"
}

func looksLikeUUID(value string) bool {
	return len(value) == 36 && strings.Count(value, "-") == 4
}

func isAllDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// WithProgress wraps base so each request in a taskgroup session is an Internet task.
// Byte progress follows Content-Length and the bytes actually read.
func WithProgress(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &progressTransport{base: base}
}
