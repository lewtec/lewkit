package progress

import (
	"context"
	"io"
	"log"
	"log/slog"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// linePrinter buffers writes and sends each '\n'-terminated line through
// print (bubbletea Program.Printf). That inserts the line above the overlay
// instead of a raw newline that only walks the cursor into the tree.
type linePrinter struct {
	print func(string)

	mu  sync.Mutex
	buf []byte
}

func (w *linePrinter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.print == nil {
		return len(p), nil
	}
	taskgroup.TakeLines(&w.buf, p, w.print)
	return len(p), nil
}

func (w *linePrinter) close() {
	w.mu.Lock()
	w.print = nil
	w.buf = nil
	w.mu.Unlock()
}

func hijackSlog(p *tea.Program) func() {
	oldSlog := slog.Default()
	oldLog := log.Default().Writer()
	w := &linePrinter{print: func(s string) { p.Printf("%s", s) }}
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: handlerLevel{oldSlog.Handler()},
	})))
	log.SetOutput(w)
	var once sync.Once
	return func() {
		once.Do(func() {
			slog.SetDefault(oldSlog)
			log.SetOutput(oldLog)
			w.close()
		})
	}
}

type handlerLevel struct{ h slog.Handler }

func (l handlerLevel) Level() slog.Level {
	for _, lv := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		if l.h.Enabled(context.Background(), lv) {
			return lv
		}
	}
	return slog.LevelInfo
}

var _ io.Writer = (*linePrinter)(nil)
