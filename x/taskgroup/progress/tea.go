//go:build !android

package progress

import (
	"bytes"
	"context"
	"os"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/lewtec/lewkit/x/taskgroup"
)

func runTea(s *taskgroup.Session, ctx context.Context, work func(context.Context) error) (err error) {
	var ui teaUI
	s.SetOnSchedule(sync.OnceFunc(ui.start(s, ctx)))
	defer s.SetOnSchedule(nil)
	defer func() {
		if uiErr := ui.stop(s); uiErr != nil && err == nil {
			err = uiErr
		}
	}()

	err = work(ctx)
	if werr := s.Wait(); err == nil {
		err = werr
	}
	return err
}

type teaUI struct {
	p       *tea.Program
	done    chan struct{}
	err     error
	restore func()
}

func (u *teaUI) start(s *taskgroup.Session, ctx context.Context) func() {
	return func() {
		m := newModel(s)
		u.p = tea.NewProgram(m, tea.WithOutput(os.Stderr), tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if _, ok := msg.(tea.InterruptMsg); ok {
				return cancelMsg{}
			}
			return msg
		}))
		s.SetLinePrint(u.print)
		u.restore = hijackSlog(ctx, u.print)
		u.done = make(chan struct{})
		go func() {
			defer close(u.done)
			_, u.err = u.p.Run()
			if ttyUnavailable(u.err) {
				u.err = nil
			}
			u.restoreLogs(s)
			if u.err != nil {
				s.Cancel(u.err)
			}
		}()
	}
}

// print uses Program.Send so a log after Run returns is a no-op.
// Program.Printf sends on p.msgs with no ctx.Done and blocks forever
// once the event loop has exited.
func (u *teaUI) print(s string) {
	if u.p == nil {
		return
	}
	u.p.Send(printLineMsg(s))
}

func (u *teaUI) restoreLogs(s *taskgroup.Session) {
	if u.restore != nil {
		u.restore()
	}
	s.SetLinePrint(nil)
}

func (u *teaUI) stop(s *taskgroup.Session) error {
	if u.p == nil {
		return nil
	}
	u.restoreLogs(s)
	for _, line := range s.TakeLiveLines() {
		_, _ = os.Stderr.WriteString(line + "\n")
	}
	select {
	case <-u.done:
	default:
		u.p.Send(doneMsg{})
		<-u.done
	}
	return u.err
}

type tickMsg time.Time

type doneMsg struct{}

type cancelMsg struct{}

type printLineMsg string

func (m model) Init() tea.Cmd {
	return m.tick()
}

func (m model) tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) refresh() {
	if m.session != nil {
		m.sync(m.session.List(m.max))
		m.live = m.session.LiveLines()
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			m.requestStop()
			return m, nil
		}
	case cancelMsg:
		m.requestStop()
		return m, nil
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		return m, nil
	case printLineMsg:
		return m, tea.Printf("%s", string(msg))
	case doneMsg:
		m.done = true
		m.refresh()
		if m.shouldQuit() {
			return m, tea.Quit
		}
		return m, m.tick()
	case tickMsg:
		m.refresh()
		if m.shouldQuit() {
			return m, tea.Quit
		}
		return m, m.tick()
	}
	return m, nil
}

func (m model) View() (view tea.View) {
	view.KeyboardEnhancements = tea.KeyboardEnhancements{}
	view.AltScreen = false
	view.MouseMode = tea.MouseModeNone
	if len(m.nodes) == 0 && len(m.live) == 0 {
		view.SetContent("")
		return
	}
	width := m.width
	if width <= 0 {
		width = defaultTermWidth
	}
	var buf bytes.Buffer
	for _, row := range m.live {
		buf.WriteString(padCells(row, width))
		buf.WriteByte('\n')
	}
	writeRows(&buf, m.nodes, width)
	view.SetContent(buf.String())
	return
}
