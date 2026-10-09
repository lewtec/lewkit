package tui

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterviewNextBackAndFinish(t *testing.T) {
	m := New([]Question{
		{Prompt: "Publisher name", Default: "Anon"},
		{Prompt: "PKCS#12 path"},
	})
	view := m.View().Content
	assert.Contains(t, view, "◆ Publisher name")
	assert.Contains(t, view, "> Anon")
	assert.Contains(t, view, "default: Anon")
	assert.Contains(t, view, "next →")
	assert.NotContains(t, view, "← back")

	m, cmd := enter(t, m)
	require.Nil(t, cmd)
	assert.Contains(t, m.View().Content, "finish")
	assert.Contains(t, m.View().Content, "← back")

	m = typeLine(t, m, "keys/a.p12")
	m, cmd = press(t, m, tea.KeyPressMsg{Code: tea.KeyLeft})
	require.Nil(t, cmd)
	assert.Contains(t, m.View().Content, "> Anon")
	assert.Contains(t, m.View().Content, "next →")

	m, cmd = enter(t, m)
	require.Nil(t, cmd)
	assert.Contains(t, m.View().Content, "> keys/a.p12")
	m, cmd = enter(t, m)
	require.NotNil(t, cmd)
	assert.True(t, m.done)
	assert.Equal(t, []Answer{{Text: "Anon"}, {Text: "keys/a.p12"}}, m.Answers())
	assert.Contains(t, m.View().Content, "◇ Publisher name")
}

func TestInterviewEnumChoices(t *testing.T) {
	m := New([]Question{{
		Prompt:  "tint",
		Default: "green",
		Choices: []string{"red", "green", "blue"},
	}})
	view := m.View().Content
	assert.Contains(t, view, "○ red")
	assert.Contains(t, view, "● green  default")
	assert.Contains(t, view, "○ blue")
	assert.Contains(t, view, "finish")

	m, cmd := press(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	require.Nil(t, cmd)
	assert.Contains(t, m.View().Content, "● red")
	m, cmd = enter(t, m)
	require.NotNil(t, cmd)
	assert.Equal(t, []Answer{{Text: "red"}}, m.Answers())
}

func TestInterviewGrowThenBackSkipsStaleStep(t *testing.T) {
	m := New([]Question{{Prompt: "path"}}).WithGrow(func(answers []Answer) ([]Question, error) {
		if answers[0].Text == "old" {
			return []Question{{
				Prompt:  "Replace old?",
				Choices: []string{"yes", "no"},
				Default: "no",
			}}, nil
		}
		return nil, nil
	})
	m = typeLine(t, m, "old")
	view := m.View().Content
	assert.Contains(t, view, "next →")
	m, cmd := enter(t, m)
	require.Nil(t, cmd)
	assert.Contains(t, m.View().Content, "● no  default")
	assert.Contains(t, m.View().Content, "finish")
	assert.Contains(t, m.View().Content, "← back")

	m, cmd = press(t, m, tea.KeyPressMsg{Code: tea.KeyLeft})
	require.Nil(t, cmd)
	for range len("old") {
		m, cmd = press(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
		require.Nil(t, cmd)
	}
	m = typeLine(t, m, "new")
	m, cmd = enter(t, m)
	require.NotNil(t, cmd)
	assert.Equal(t, []Answer{{Text: "new"}}, m.Answers())
}

func TestInterviewGrowError(t *testing.T) {
	boom := errors.New("bad path")
	m := New([]Question{{Prompt: "path"}}).WithGrow(func([]Answer) ([]Question, error) {
		return nil, boom
	})
	m, cmd := enter(t, m)
	require.NotNil(t, cmd)
	assert.ErrorIs(t, m.fail, boom)
}

func TestInterviewBackspaceAndEmptyLine(t *testing.T) {
	m := New([]Question{{Prompt: "Publisher name"}})
	m = typeLine(t, m, "A ")
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeySpace})
	m, cmd := enter(t, m)
	require.NotNil(t, cmd)
	assert.Equal(t, []Answer{{Text: "A "}}, m.Answers())
}

func TestInterviewCancel(t *testing.T) {
	m := New([]Question{{Prompt: "Publisher name"}})
	m = typeLine(t, m, "Acme")
	m, cmd := press(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	assert.True(t, m.canceled)
	assert.Empty(t, m.Answers())
}

func TestRunNilContext(t *testing.T) {
	_, err := Run(nil, New([]Question{{Prompt: "Publisher name"}}))
	require.ErrorIs(t, err, ErrNilContext)
}

func TestRunCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := Run(ctx, New([]Question{{Prompt: "Publisher name"}}))
	require.ErrorIs(t, err, context.Canceled)
}

func TestRunEmpty(t *testing.T) {
	answers, err := Run(t.Context(), New(nil))
	require.NoError(t, err)
	assert.Nil(t, answers)
}

func TestRunProgram(t *testing.T) {
	var in, out bytes.Buffer
	_, _ = in.WriteString("Acme\rkeys/a.p12\r")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	answers, err := start(ctx, New([]Question{
		{Prompt: "Publisher name"},
		{Prompt: "PKCS#12 path"},
	}), &in, &out)
	require.NoError(t, err)
	assert.Equal(t, []Answer{
		{Text: "Acme"},
		{Text: "keys/a.p12"},
	}, answers)
}

func typeLine(t *testing.T, m Interview, text string) Interview {
	t.Helper()
	for _, r := range text {
		msg := tea.KeyPressMsg{Text: string(r), Code: r}
		if r == ' ' {
			msg = tea.KeyPressMsg{Code: tea.KeySpace}
		}
		var cmd tea.Cmd
		m, cmd = press(t, m, msg)
		require.Nil(t, cmd)
	}
	return m
}

func enter(t *testing.T, m Interview) (Interview, tea.Cmd) {
	t.Helper()
	return press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
}

func press(t *testing.T, m Interview, msg tea.KeyPressMsg) (Interview, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	got, ok := next.(Interview)
	require.True(t, ok)
	return got, cmd
}
