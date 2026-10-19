package term

import "time"

// cwdPoll is how often a live pane's process directory is read.
// cd in the shell changes that process and does not have to print anything.
const cwdPoll = 200 * time.Millisecond

func (s *Session) watchCwd(id string) {
	if !s.pollCwd(id) {
		return
	}
	tick := time.NewTicker(cwdPoll)
	defer tick.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-tick.C:
			if !s.pollCwd(id) {
				return
			}
		}
	}
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
