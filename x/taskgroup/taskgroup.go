// Package taskgroup is a dependency-aware executor with resource pools
// and a live task tree.
//
// New starts a Session. Flatten-embed Arg on a cmd spec for --io/--cpu/
// --internet; Arg.Value is the Session after Enter. Go, Map, and Each attach
// nodes under the current task (from ctx). List(n) walks live children
// until it has n rows; it does not flatten the forest first.
//
// Only leaf tasks take IO, CPU, or Internet. Orchestrators are Control.
// Isolate is an error boundary with no list row.
package taskgroup

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
)

var (
	ErrUnknownDependency = errors.New("unknown dependency")
	ErrDependencyFailed  = errors.New("dependency failed")
	ErrNilFn             = errors.New("fn is nil")
)

// PoolKind identifies which resource pool a task consumes a slot from.
// The zero value is Control, the default for orchestrators.
type PoolKind int

const (
	Control  PoolKind = iota // Unlimited orchestrator; wrap nested work
	IO                       // Leaf: local disk
	CPU                      // Leaf: computation / local exec
	Internet                 // Leaf: network
)

func (p PoolKind) String() string {
	switch p {
	case Control:
		return "control"
	case IO:
		return "io"
	case CPU:
		return "cpu"
	case Internet:
		return "internet"
	default:
		return "unknown"
	}
}

// Limits holds the concurrency limits for each leaf pool.
// A zero field in New means that pool uses DefaultLimits.
type Limits struct {
	IO       int
	CPU      int
	Internet int
}

// DefaultLimits returns IO=4, CPU=NumCPU, Internet=4.
func DefaultLimits() Limits {
	return Limits{
		IO:       4,
		CPU:      max(runtime.NumCPU(), 1),
		Internet: 4,
	}
}

func (l Limits) norm() Limits {
	d := DefaultLimits()
	if l.IO <= 0 {
		l.IO = d.IO
	}
	if l.CPU <= 0 {
		l.CPU = d.CPU
	}
	if l.Internet <= 0 {
		l.Internet = d.Internet
	}
	return l
}

// State is the lifecycle of a node.
type State int

const (
	Pending State = iota
	Running
	Done
	Failed
)

func (s State) String() string {
	switch s {
	case Pending:
		return "pending"
	case Running:
		return "running"
	case Done:
		return "done"
	case Failed:
		return "failed"
	default:
		return "unknown"
	}
}

// ID is a dense node key for one Session. Zero is not a task.
type ID uint32

// Node is a point-in-time view of one tree node.
// List walks live links and stops at n; it does not copy the whole tree.
type Node struct {
	ID           ID
	Parent       ID
	Name         string
	Pool         PoolKind
	State        State
	Message      string
	Current      int64
	Total        int64 // -1 means indeterminate
	LiveChildren int
}

// Status is the handle given to a task function so it can report progress.
type Status struct {
	t *task
}

// Update sets the status message shown on this node.
func (s *Status) Update(message string) {
	if s == nil || s.t == nil {
		return
	}
	s.t.message.Store(&message)
}

// Progress sets the current/total counters. Use total=-1 for indeterminate.
func (s *Status) Progress(current, total int64) {
	if s == nil || s.t == nil {
		return
	}
	s.t.current.Store(current)
	s.t.total.Store(total)
}

// Unit marks this task as a single-step item (0/1).
// Returns a done func that sets 1/1; call via defer s.Unit()().
//
// Skip Unit when a nested task already owns real progress (Map aggregate).
func (s *Status) Unit() (done func()) {
	s.Progress(0, 1)
	return func() { s.Progress(1, 1) }
}

type sessionKey struct{}
type taskKey struct{}

// Session owns the task tree and the worker pools for one run.
type Session struct {
	ctx    context.Context
	cancel context.CancelCauseFunc

	mu   sync.Mutex
	cond *sync.Cond

	slots        []*task
	root         ID
	latestByDesc map[string]ID

	ready [4]*readyQ

	workers sync.WaitGroup
	tasks   sync.WaitGroup

	err  error
	hard atomic.Bool

	live    *liveHub
	printMu sync.Mutex
	print   func(string)
}

type task struct {
	id, parent                              ID
	firstChild, lastChild, prevSib, nextSib ID
	firstLive, lastLive, prevLive, nextLive ID

	name    string
	pool    PoolKind
	isolate bool

	state   atomic.Uint32
	current atomic.Int64
	total   atomic.Int64
	message atomic.Pointer[string]
	liveN   atomic.Int32

	remainingDeps uint32
	waiters       []ID
	fn            func(context.Context, *Status) error
	err           error

	ctx    context.Context
	cancel context.CancelCauseFunc
}

// New creates a Session with the given pool limits.
// The returned context carries the Session and is cancelled on first
// non-isolated task error.
func New(ctx context.Context, limits Limits) (*Session, context.Context) {
	limits = limits.norm()
	ctx, cancel := context.WithCancelCause(ctx)
	root := &task{id: 1, ctx: ctx}
	root.total.Store(-1)
	s := &Session{
		ctx:          ctx,
		cancel:       cancel,
		slots:        []*task{nil, root},
		root:         1,
		latestByDesc: make(map[string]ID),
		live:         newLiveHub(),
	}
	s.cond = sync.NewCond(&s.mu)
	s.ready[IO] = newReadyQ()
	s.ready[CPU] = newReadyQ()
	s.ready[Internet] = newReadyQ()
	s.startWorkers(limits)
	context.AfterFunc(ctx, func() {
		go func() {
			s.mu.Lock()
			if s.hard.Load() || errors.Is(context.Cause(s.ctx), context.Canceled) {
				s.failSubtree(s.root)
			} else {
				s.failOutsideIsolates(s.root)
			}
			s.mu.Unlock()
		}()
		for _, q := range s.ready {
			if q != nil {
				q.wake()
			}
		}
		s.cond.Broadcast()
	})
	return s, context.WithValue(ctx, sessionKey{}, s)
}

// Cancel stops the session. Pending work fails; running tasks see ctx cancel.
func (s *Session) Cancel(err error) {
	if err == nil {
		err = context.Canceled
	}
	s.hard.Store(true)
	s.cancel(err)
}

func (s *Session) startWorkers(limits Limits) {
	start := func(kind PoolKind, n int) {
		q := s.ready[kind]
		for range n {
			s.workers.Go(func() {
				for {
					id, ok := q.pop(s.ctx)
					if !ok {
						return
					}
					s.run(id)
				}
			})
		}
	}
	start(IO, limits.IO)
	start(CPU, limits.CPU)
	start(Internet, limits.Internet)
}

// FromContext retrieves the Session from ctx. Returns nil if none.
func FromContext(ctx context.Context) *Session {
	s, ok := ctx.Value(sessionKey{}).(*Session)
	if !ok {
		return nil
	}
	return s
}

// MustFromContext is like FromContext but panics if no Session is present.
func MustFromContext(ctx context.Context) *Session {
	if s := FromContext(ctx); s != nil {
		return s
	}
	panic("taskgroup: no Session present in context; " +
		"only the top-level caller may call New, everything else must receive it via context")
}

func taskFromContext(ctx context.Context) ID {
	id, ok := ctx.Value(taskKey{}).(ID)
	if !ok {
		return 0
	}
	return id
}

// Wait blocks until every scheduled task finishes and then stops the workers.
// It returns the first non-isolated error, if any.
func (s *Session) Wait() error {
	s.tasks.Wait()
	s.mu.Lock()
	err := s.err
	s.mu.Unlock()
	s.cancel(err)
	for _, q := range s.ready {
		if q != nil {
			q.close()
		}
	}
	s.workers.Wait()
	return err
}

// Latest returns the most recently scheduled live task with that name, or 0.
func (s *Session) Latest(name string) ID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latestByDesc[name]
}
