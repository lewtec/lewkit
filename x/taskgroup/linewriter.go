package taskgroup

import (
	"context"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

// liveHub holds the in-progress (not yet newline-terminated) row for each
// LineWriter. The progress view snapshots this every tick.
type liveHub struct {
	mu      sync.Mutex
	seq     atomic.Uint64
	order   []string
	slots   map[string]string
	writers map[string]*lineWriter
}

func newLiveHub() *liveHub {
	return &liveHub{
		slots:   make(map[string]string),
		writers: make(map[string]*lineWriter),
	}
}

func (h *liveHub) snapshot() []string {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.order))
	for _, id := range h.order {
		if text, ok := h.slots[id]; ok && text != "" {
			out = append(out, text)
		}
	}
	return out
}

func (h *liveHub) set(id, text string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if text == "" {
		h.dropLocked(id)
		return
	}
	if _, ok := h.slots[id]; !ok {
		h.order = append(h.order, id)
	}
	h.slots[id] = text
}

func (h *liveHub) drop(id string) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.dropLocked(id)
}

func (h *liveHub) dropLocked(id string) {
	if _, ok := h.slots[id]; !ok {
		return
	}
	delete(h.slots, id)
	for i, x := range h.order {
		if x == id {
			h.order = append(h.order[:i], h.order[i+1:]...)
			break
		}
	}
}

func (h *liveHub) register(w *lineWriter) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.writers[w.id] = w
	h.mu.Unlock()
}

func (h *liveHub) abandonAll() []string {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	ws := make([]*lineWriter, 0, len(h.writers))
	for _, w := range h.writers {
		ws = append(ws, w)
	}
	h.writers = make(map[string]*lineWriter)
	h.slots = make(map[string]string)
	h.order = nil
	h.mu.Unlock()

	var leftover []string
	for _, w := range ws {
		if s := w.abandon(); s != "" {
			leftover = append(leftover, s)
		}
	}
	return leftover
}

type lineWriter struct {
	id            string
	hub           *liveHub
	print         func(string)
	commitOnClose bool

	mu     sync.Mutex
	filter lineFilter
	closed bool
}

func newLineWriter(hub *liveHub, print func(string)) *lineWriter {
	w := &lineWriter{
		hub:           hub,
		print:         print,
		commitOnClose: true,
	}
	if hub != nil {
		w.id = "live-" + strconv.FormatUint(hub.seq.Add(1), 10)
		hub.register(w)
	}
	return w
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	committed := w.filter.write(p)
	cur := w.filter.current()
	print := w.print
	id := w.id
	hub := w.hub
	w.mu.Unlock()

	for _, line := range committed {
		if print != nil {
			print(line)
		}
	}
	hub.set(id, cur)
	return len(p), nil
}

// ReadFrom is used by os/exec when Stderr is not an *os.File.
func (w *lineWriter) ReadFrom(r io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var n int64
	for {
		nr, err := r.Read(buf)
		if nr > 0 {
			nw, werr := w.Write(buf[:nr])
			n += int64(nw)
			if werr != nil {
				_ = w.Close()
				return n, werr
			}
		}
		if err != nil {
			cErr := w.Close()
			if err == io.EOF {
				return n, cErr
			}
			if cErr != nil {
				return n, cErr
			}
			return n, err
		}
	}
}

func (w *lineWriter) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	text := w.filter.take()
	print := w.print
	flush := w.commitOnClose
	id := w.id
	hub := w.hub
	w.mu.Unlock()

	hub.drop(id)
	if flush && text != "" && print != nil {
		print(text)
	}
	return nil
}

func (w *lineWriter) abandon() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return ""
	}
	w.closed = true
	w.print = nil
	return w.filter.take()
}

// LineWriterFrom returns a writer for one subprocess stream.
//
// With an active progress UI the writer is a live row: CR / erase-line
// rewrite that row until a newline commits it above the tree. Close
// commits leftover. Without a session (or no UI) only finished lines
// go to os.Stderr.
func LineWriterFrom(ctx context.Context) io.WriteCloser {
	if s := FromContext(ctx); s != nil {
		return s.LineWriter()
	}
	return newFinishedLineWriter(os.Stderr)
}

// LineWriter returns a new live-row writer for this session.
// Safe to assign to cmd.Stdout and cmd.Stderr (same writer for both).
func (s *Session) LineWriter() io.WriteCloser {
	if s == nil || s.live == nil {
		return newFinishedLineWriter(os.Stderr)
	}
	fn := s.linePrintFn()
	if fn == nil {
		return newFinishedLineWriter(os.Stderr)
	}
	return newLineWriter(s.live, fn)
}

// SetLinePrint is the sink for committed lines while the progress TUI is up.
// Pass nil to stop.
func (s *Session) SetLinePrint(fn func(string)) {
	if s == nil {
		return
	}
	s.printMu.Lock()
	s.print = fn
	s.printMu.Unlock()
}

func (s *Session) linePrintFn() func(string) {
	if s == nil {
		return nil
	}
	s.printMu.Lock()
	defer s.printMu.Unlock()
	return s.print
}

// LiveLines is the current uncommitted CR row for each open LineWriter.
func (s *Session) LiveLines() []string {
	if s == nil {
		return nil
	}
	return s.live.snapshot()
}

// TakeLiveLines closes leftover writers and returns their last text.
func (s *Session) TakeLiveLines() []string {
	if s == nil {
		return nil
	}
	return s.live.abandonAll()
}

func newFinishedLineWriter(out io.Writer) *lineWriter {
	return &lineWriter{
		print: func(s string) {
			_, _ = io.WriteString(out, s+"\n")
		},
	}
}
