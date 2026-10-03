package app

import (
	"context"
	"errors"
	"sync"
)

// errAppRunning is returned when Run starts while a session is still open.
var errAppRunning = errors.New("app is already running")

// windows is the process session. Callers pass Open the context they captured
// before Run, so the session cannot live only on a context created inside Run.
type windows struct {
	ctx     context.Context
	cancel  context.CancelFunc
	profile string
	mu      sync.Mutex
	n       int
	done    chan struct{}
	once    sync.Once
}

var (
	appMu      sync.Mutex
	appWindows *windows
)

func bindSession(ctx context.Context, profile string) (*windows, error) {
	appMu.Lock()
	defer appMu.Unlock()
	if appWindows != nil {
		return nil, errAppRunning
	}
	child, cancel := context.WithCancel(ctx)
	s := &windows{
		ctx:     child,
		cancel:  cancel,
		profile: profile,
		done:    make(chan struct{}),
	}
	appWindows = s
	return s, nil
}

func (s *windows) unbind() {
	appMu.Lock()
	if appWindows == s {
		appWindows = nil
	}
	appMu.Unlock()
	s.cancel()
}

func currentSession() *windows {
	appMu.Lock()
	defer appMu.Unlock()
	return appWindows
}

func (s *windows) add() {
	s.mu.Lock()
	s.n++
	s.mu.Unlock()
}

func (s *windows) remove() {
	s.mu.Lock()
	s.n--
	last := s.n == 0
	s.mu.Unlock()
	if last {
		s.once.Do(func() {
			s.cancel()
			close(s.done)
		})
	}
}

// openCall is one window in the session.
type openCall struct {
	title   string
	width   int
	height  int
	profile string
}

// runSession opens the first window and returns when the last window closes.
// The first window's error is that result. Canceling ctx closes every window.
func runSession(ctx context.Context, win Window, call openCall) error {
	s, err := bindSession(ctx, call.profile)
	if err != nil {
		return err
	}
	defer s.unbind()
	s.add()
	openErr := win.open(s.ctx, call.title, call.width, call.height, call.profile)
	s.remove()
	select {
	case <-s.done:
	case <-ctx.Done():
		s.cancel()
		<-s.done
	}
	return openErr
}

func (s *windows) open(ctx context.Context, w Window, title string, width, height int) error {
	wctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	if ctx != nil {
		stop := context.AfterFunc(ctx, cancel)
		defer stop()
	}
	s.add()
	defer s.remove()
	return w.open(wctx, title, width, height, s.profile)
}
