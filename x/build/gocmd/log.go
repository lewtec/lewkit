package gocmd

import (
	"bytes"
	"io"
	"log/slog"
	"sync"
)

// LogWriter sends each newline-terminated line to slog.Info.
// The progress view hijacks slog, so these lines stay above the tree.
func LogWriter() io.Writer { return &slogWriter{} }

type slogWriter struct {
	mu  sync.Mutex
	buf []byte
}

func (w *slogWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := bytes.TrimSpace(w.buf[:i])
		w.buf = w.buf[i+1:]
		if len(line) > 0 {
			slog.Info(string(line))
		}
	}
	return len(p), nil
}

func (w *slogWriter) flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	line := bytes.TrimSpace(w.buf)
	w.buf = nil
	if len(line) > 0 {
		slog.Info(string(line))
	}
}
