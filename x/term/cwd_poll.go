package term

import "time"

// cwdQuiet is how long a pane's output must stay quiet before its
// process directory is read. More output starts the wait over.
// An idle pane is not read again until the next output.
const cwdQuiet = 200 * time.Millisecond

func (s *Session) watchCwd(id string, output <-chan struct{}) {
	var quiet <-chan time.Time
	var timer *time.Timer
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-s.ctx.Done():
			return
		case _, ok := <-output:
			if !ok {
				return
			}
			timer = armQuiet(timer)
			quiet = timer.C
		case <-quiet:
			quiet = nil
			if !s.pollCwd(id) {
				return
			}
		}
	}
}

func armQuiet(timer *time.Timer) *time.Timer {
	if timer == nil {
		return time.NewTimer(cwdQuiet)
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(cwdQuiet)
	return timer
}

func noteOutput(pane *live) {
	if pane == nil || pane.output == nil || pane.outClosed {
		return
	}
	pane.sawOutput = true
	select {
	case pane.output <- struct{}{}:
	default:
	}
}

func (s *Session) stopOutput(pane *live) {
	if pane == nil || pane.output == nil || pane.outClosed {
		return
	}
	pane.outClosed = true
	close(pane.output)
}

func (s *Session) pollCwd(id string) bool {
	s.mu.Lock()
	pane, ok := s.panes[id]
	if !ok || s.closed || pane.proc == nil {
		s.mu.Unlock()
		return false
	}
	if !s.observeCwd(pane) {
		s.mu.Unlock()
		return true
	}
	fns := s.publishLocked(pane, true)
	s.mu.Unlock()
	s.saveState()
	deliver(fns)
	return true
}
