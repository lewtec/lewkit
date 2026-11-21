package taskgroup

import (
	"context"
	"sync"
)

// readyQ is a cond-backed deque. A buffered chan of size Limits would
// park every pending leaf; this holds IDs only.
type readyQ struct {
	mu     sync.Mutex
	cond   *sync.Cond
	q      []ID
	closed bool
}

func newReadyQ() *readyQ {
	q := &readyQ{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *readyQ) push(id ID) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.q = append(q.q, id)
	q.cond.Signal()
}

func (q *readyQ) pop(ctx context.Context) (ID, bool) {
	stop := context.AfterFunc(ctx, q.wake)
	defer stop()

	q.mu.Lock()
	defer q.mu.Unlock()
	for len(q.q) == 0 && !q.closed && ctx.Err() == nil {
		q.cond.Wait()
	}
	if len(q.q) == 0 {
		return 0, false
	}
	id := q.q[0]
	q.q = q.q[1:]
	return id, true
}

func (q *readyQ) wake() {
	q.mu.Lock()
	q.cond.Broadcast()
	q.mu.Unlock()
}

func (q *readyQ) close() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()
}
