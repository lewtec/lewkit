package androidask

import (
	"sync"

	"github.com/lewtec/lewkit/x/errs"
)

// ErrBusy means a dialog is already waiting.
var ErrBusy = errs.Wrap(errs.ErrAlreadyExists, "dialog already open")

// Pending is one in-flight Android callback.
// Begin fails with busy while a call is open. Deliver drops a stale generation
// and a second value for the same generation.
type Pending[T any] struct {
	busy error

	mu  sync.Mutex
	cur *pendingBox[T]
	gen uint64
}

type pendingBox[T any] struct {
	ch  chan T
	gen uint64
}

// NewPending returns a slot that reports busy from Begin while one call is open.
func NewPending[T any](busy error) *Pending[T] {
	return &Pending[T]{busy: busy}
}

// Begin opens the slot. A second call before End returns busy.
func (p *Pending[T]) Begin() (<-chan T, uint64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cur != nil {
		return nil, 0, p.busy
	}
	p.gen++
	ch := make(chan T, 1)
	p.cur = &pendingBox[T]{ch: ch, gen: p.gen}
	return ch, p.gen, nil
}

// End closes the slot so Begin can run again.
func (p *Pending[T]) End() {
	p.mu.Lock()
	p.cur = nil
	p.mu.Unlock()
}

// Deliver sends value when gen is the open call. A full channel keeps the first value.
func (p *Pending[T]) Deliver(gen uint64, value T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cur == nil || p.cur.gen != gen {
		return
	}
	select {
	case p.cur.ch <- value:
	default:
	}
}

var asks = NewPending[string](ErrBusy)

func begin() (<-chan string, uint64, error) { return asks.Begin() }

func end() { asks.End() }

func deliver(g uint64, text string) { asks.Deliver(g, text) }
