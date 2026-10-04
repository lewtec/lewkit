package androidask

import (
	"errors"
	"sync"
)

// ErrBusy means a dialog is already waiting.
var ErrBusy = errors.New("dialog already open")

type box struct {
	ch  chan string
	gen uint64
}

var (
	waitMu sync.Mutex
	cur    *box
	gen    uint64
)

func begin() (<-chan string, uint64, error) {
	waitMu.Lock()
	defer waitMu.Unlock()
	if cur != nil {
		return nil, 0, ErrBusy
	}
	gen++
	ch := make(chan string, 1)
	cur = &box{ch: ch, gen: gen}
	return ch, gen, nil
}

func end() {
	waitMu.Lock()
	cur = nil
	waitMu.Unlock()
}

func deliver(g uint64, text string) {
	waitMu.Lock()
	defer waitMu.Unlock()
	if cur == nil || cur.gen != g {
		return
	}
	select {
	case cur.ch <- text:
	default:
	}
}
