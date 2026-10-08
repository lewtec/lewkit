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

// Question is one step.
// Choices, when set, are a closed list. Otherwise the answer is the entered line.
// Default is the initial value. On a choice list it is the selected item.
type Question struct {
	Prompt  string
	Default string
	Choices []string
}

// Answer is the reply to one question, in order.
type Answer struct {
	Text string
	Yes  bool
}

// Grow adds questions when the interview advances past the last original step.
// A nil slice finishes the interview.
type Grow func(answers []Answer) ([]Question, error)

// Interview asks one question at a time.
// Left is back. Enter and right are next, and finish on the last step.
// Run starts it with the caller context. A nil context is ErrNilContext.
type Interview struct {
	questions []Question
	answers   []Answer
	input     []rune
	index     int
	choice    int
	base      int
	grow      Grow
	done      bool
	canceled  bool
	fail      error
}

// New copies questions into an interview and shows the first default.
func New(questions []Question) Interview {
	qs := make([]Question, len(questions))
	for i, q := range questions {
		qs[i] = q
		if q.Choices != nil {
			qs[i].Choices = append([]string(nil), q.Choices...)
		}
	}
	m := Interview{questions: qs, base: len(qs)}
	if len(qs) > 0 {
		m = m.load()
	}
	return m
}

// WithGrow sets the hook that can append steps before finish.
func (m Interview) WithGrow(grow Grow) Interview {
	m.grow = grow
	return m
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
	if m.done || m.canceled || m.fail != nil || len(m.questions) == 0 || m.index >= len(m.questions) {
		m.done = len(m.questions) > 0 && m.fail == nil
		return m, tea.Quit
	}
	switch msg.String() {
	case "ctrl+c", "esc":
		m.canceled = true
		return m, tea.Quit
	case "left":
		return m.back(), nil
	case "right", "enter":
		return m.forward()
	case "up":
		return m.move(-1), nil
	case "down":
		return m.move(1), nil
	}
	if len(m.questions[m.index].Choices) > 0 {
		return m, nil
	}
	return m.text(msg)
}

func (m Interview) text(msg tea.KeyPressMsg) (Interview, tea.Cmd) {
	switch msg.String() {
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

func (m Interview) move(d int) Interview {
	q := m.questions[m.index]
	if len(q.Choices) == 0 {
		return m
	}
	n := m.choice + d
	if n < 0 || n >= len(q.Choices) {
		return m
	}
	m.choice = n
	return m
}

func (m Interview) back() Interview {
	if m.index == 0 {
		return m
	}
	m = m.save()
	m.index--
	return m.load()
}

func (m Interview) forward() (Interview, tea.Cmd) {
	m = m.save()
	if m.grow != nil && m.index == m.base-1 {
		extra, err := m.grow(m.answers)
		if err != nil {
			m.fail = err
			return m, tea.Quit
		}
		m.questions = append(append([]Question{}, m.questions[:m.base]...), extra...)
		if len(extra) > 0 {
			m.index = m.base
			return m.load(), nil
		}
		m.done = true
		m.answers = append([]Answer(nil), m.answers[:m.base]...)
		return m, tea.Quit
	}
	if m.index+1 < len(m.questions) {
		m.index++
		return m.load(), nil
	}
	m.done = true
	m.answers = append([]Answer(nil), m.answers[:m.index+1]...)
	return m, tea.Quit
}

func (m Interview) current() Answer {
	q := m.questions[m.index]
	if len(q.Choices) > 0 {
		if m.choice < 0 || m.choice >= len(q.Choices) {
			return Answer{}
		}
		text := q.Choices[m.choice]
		return Answer{Text: text, Yes: text == "yes"}
	}
	return Answer{Text: string(m.input)}
}

func (m Interview) save() Interview {
	ans := m.current()
	if m.index < len(m.answers) {
		m.answers[m.index] = ans
		return m
	}
	for len(m.answers) < m.index {
		m.answers = append(m.answers, Answer{})
	}
	m.answers = append(m.answers, ans)
	return m
}

func (m Interview) load() Interview {
	q := m.questions[m.index]
	text := q.Default
	if m.index < len(m.answers) {
		text = m.answers[m.index].Text
	}
	m.input = nil
	m.choice = 0
	if len(q.Choices) > 0 {
		for i, choice := range q.Choices {
			if choice == text {
				m.choice = i
				break
			}
		}
		return m
	}
	m.input = []rune(text)
	return m
}

func (m Interview) preview() []Answer {
	m.answers = append([]Answer(nil), m.answers...)
	return m.save().answers
}

func (m Interview) forwardLabel() string {
	if m.grow != nil && m.index == m.base-1 {
		extra, err := m.grow(m.preview())
		if err == nil && len(extra) > 0 {
			return "next →"
		}
		return "finish"
	}
	if m.index < len(m.questions)-1 {
		return "next →"
	}
	return "finish"
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
		if i < m.index {
			b.WriteString(styleDone.Render("◇ " + q.Prompt))
			b.WriteByte('\n')
			b.WriteString(styleDone.Render("│ " + answerText(q, m.answers, i)))
			continue
		}
		b.WriteString(styleActive.Render("◆ " + q.Prompt))
		b.WriteByte('\n')
		if len(q.Choices) > 0 {
			for ci, choice := range q.Choices {
				if ci > 0 {
					b.WriteByte('\n')
				}
				row := "│ "
				if ci == m.choice {
					row += "● "
				} else {
					row += "○ "
				}
				row += choice
				if q.Default != "" && choice == q.Default {
					row += "  default"
				}
				if ci == m.choice {
					b.WriteString(stylePick.Render(row))
				} else {
					b.WriteString(styleChoice.Render(row))
				}
			}
		} else {
			line := string(m.input)
			plain := "│ > " + line
			b.WriteString(styleInput.Render(plain))
			showCursor = true
			cursorX = runewidth.StringWidth(plain)
			cursorY = strings.Count(b.String(), "\n")
			if q.Default != "" {
				b.WriteByte('\n')
				b.WriteString(styleHint.Render("│ default: " + q.Default))
			}
		}
	}
	if len(m.questions) > 0 && m.index < len(m.questions) {
		b.WriteByte('\n')
		b.WriteString(styleBar.Render("│"))
		b.WriteByte('\n')
		b.WriteString(styleBar.Render("│ "))
		if m.index > 0 {
			b.WriteString(styleBack.Render("← back"))
			b.WriteString(styleBar.Render("    "))
		}
		b.WriteString(styleNext.Render(m.forwardLabel()))
		b.WriteByte('\n')
		b.WriteString(styleBar.Render("└"))
	}
	view := tea.NewView(b.String())
	if showCursor {
		view.Cursor = tea.NewCursor(cursorX, cursorY)
	}
	return view
}

func answerText(q Question, answers []Answer, i int) string {
	if i >= len(answers) {
		return ""
	}
	return answers[i].Text
}

// Run asks questions on the caller context and returns each answer.
// An empty interview returns no answers. Cancel and ctrl+c are ErrCanceled.
func Run(ctx context.Context, m Interview) ([]Answer, error) {
	return start(ctx, m, nil, os.Stderr)
}

func start(ctx context.Context, m Interview, in io.Reader, out io.Writer) ([]Answer, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(m.questions) == 0 {
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
	final, err := tea.NewProgram(m, opts...).Run()
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
	if !ok || done.canceled {
		return nil, ErrCanceled
	}
	if done.fail != nil {
		return nil, done.fail
	}
	if !done.done {
		return nil, ErrCanceled
	}
	return done.Answers(), nil
}
