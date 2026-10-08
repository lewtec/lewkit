// Package tui holds bubbletea types for callers to compose.
// A Session viewer stays next to that session.
package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
)

// ErrNilContext means the caller did not pass a context.
var ErrNilContext = errors.New("tui: nil context")

// ErrCanceled means the interview stopped before every question had an answer.
var ErrCanceled = errors.New("tui: canceled")

// Question is one step. Confirm is answered with y or n.
// Any other question records the entered line.
type Question struct {
	Prompt  string
	Confirm bool
}

// Answer is the reply to one question, in order.
type Answer struct {
	Text string
	Yes  bool
}

// Interview asks each question in order.
// Run starts it with the caller context. A nil context is ErrNilContext.
type Interview struct {
	questions []Question
	answers   []Answer
	input     []rune
	index     int
	done      bool
	canceled  bool
}

// New copies questions into an interview.
func New(questions []Question) Interview {
	return Interview{questions: append([]Question(nil), questions...)}
}

// Answers returns the replies recorded so far.
func (m Interview) Answers() []Answer {
	return append([]Answer(nil), m.answers...)
}

// Init implements tea.Model.
func (m Interview) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Interview) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.key(msg)
	default:
		return m, nil
	}
}

func (m Interview) key(msg tea.KeyPressMsg) (Interview, tea.Cmd) {
	if m.done || m.canceled || m.index >= len(m.questions) {
		m.done = m.index >= len(m.questions) && len(m.questions) > 0
		return m, tea.Quit
	}
	switch msg.String() {
	case "ctrl+c", "esc":
		m.canceled = true
		return m, tea.Quit
	}
	if m.questions[m.index].Confirm {
		return m.confirm(msg)
	}
	return m.text(msg)
}

func (m Interview) confirm(msg tea.KeyPressMsg) (Interview, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		return m.commit(Answer{Text: "y", Yes: true})
	case "n":
		return m.commit(Answer{Text: "n", Yes: false})
	default:
		return m, nil
	}
}

func (m Interview) text(msg tea.KeyPressMsg) (Interview, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m.commit(Answer{Text: string(m.input)})
	case "backspace":
		if n := len(m.input); n > 0 {
			m.input = m.input[:n-1]
		}
		return m, nil
	case "space":
		m.input = append(m.input, ' ')
		return m, nil
	}
	if text := msg.Text; text != "" && msg.String() == text {
		mod := msg.Key().Mod
		if mod.Contains(tea.ModCtrl) || mod.Contains(tea.ModAlt) {
			return m, nil
		}
		m.input = append(m.input, []rune(text)...)
	}
	return m, nil
}

func (m Interview) commit(answer Answer) (Interview, tea.Cmd) {
	m.answers = append(m.answers, answer)
	m.input = nil
	m.index++
	if m.index >= len(m.questions) {
		m.done = true
		return m, tea.Quit
	}
	return m, nil
}

// View implements tea.Model.
func (m Interview) View() tea.View {
	var b strings.Builder
	cursorX, cursorY := 0, 0
	showCursor := false
	for i, q := range m.questions {
		if i > m.index {
			break
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(q.Prompt)
		b.WriteByte('\n')
		b.WriteString("> ")
		line := ""
		if i < m.index && i < len(m.answers) {
			line = m.answers[i].Text
			if q.Confirm {
				line = "n"
				if m.answers[i].Yes {
					line = "y"
				}
			}
		}
		if i == m.index {
			line = string(m.input)
			showCursor = true
			cursorX = 2 + runewidth.StringWidth(line)
			cursorY = strings.Count(b.String(), "\n")
		}
		b.WriteString(line)
	}
	view := tea.NewView(b.String())
	if showCursor {
		view.Cursor = tea.NewCursor(cursorX, cursorY)
	}
	return view
}

// Run asks questions on the caller context and returns each answer.
// An empty list returns no answers. Cancel and ctrl+c are ErrCanceled.
func Run(ctx context.Context, questions []Question) ([]Answer, error) {
	return run(ctx, questions, nil, os.Stderr)
}

func run(ctx context.Context, questions []Question, in io.Reader, out io.Writer) ([]Answer, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(questions) == 0 {
		return nil, nil
	}
	if out == nil {
		out = os.Stderr
	}
	opts := []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithOutput(out),
	}
	if in != nil {
		opts = append(opts, tea.WithInput(in))
	}
	final, err := tea.NewProgram(New(questions), opts...).Run()
	if errors.Is(err, tea.ErrProgramKilled) || errors.Is(err, tea.ErrInterrupted) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrCanceled
	}
	if err != nil {
		return nil, err
	}
	done, ok := final.(Interview)
	if !ok || done.canceled || !done.done {
		return nil, ErrCanceled
	}
	return done.Answers(), nil
}
