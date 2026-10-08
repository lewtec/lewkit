package tui

import (
	"bytes"
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterviewRecordsTextAndConfirm(t *testing.T) {
	m := New([]Question{
		{Prompt: "Publisher name"},
		{Prompt: "PKCS#12 path"},
		{Prompt: "Replace key?", Confirm: true},
	})
	m = typeLine(t, m, "Acme")
	m, cmd := enter(t, m)
	require.Nil(t, cmd)
	m = typeLine(t, m, "keys/publisher.p12")
	m, cmd = enter(t, m)
	require.Nil(t, cmd)
	m, cmd = press(t, m, tea.KeyPressMsg{Text: "y", Code: 'y'})
	require.NotNil(t, cmd)
	assert.True(t, m.done)
	assert.Equal(t, []Answer{
		{Text: "Acme"},
		{Text: "keys/publisher.p12"},
		{Text: "y", Yes: true},
	}, m.Answers())
	view := m.View().Content
	assert.Contains(t, view, "Publisher name")
	assert.Contains(t, view, "> Acme")
	assert.Contains(t, view, "> y")
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

func TestInterviewRejectsOtherConfirmKeys(t *testing.T) {
	m := New([]Question{{Prompt: "Replace?", Confirm: true}})
	m, cmd := enter(t, m)
	require.Nil(t, cmd)
	assert.False(t, m.done)
	m, cmd = press(t, m, tea.KeyPressMsg{Text: "n", Code: 'n'})
	require.NotNil(t, cmd)
	assert.Equal(t, []Answer{{Text: "n", Yes: false}}, m.Answers())
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
	_, err := Run(nil, []Question{{Prompt: "Publisher name"}})
	require.ErrorIs(t, err, ErrNilContext)
}

func TestRunCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := Run(ctx, []Question{{Prompt: "Publisher name"}})
	require.ErrorIs(t, err, context.Canceled)
}

func TestRunEmpty(t *testing.T) {
	answers, err := Run(t.Context(), nil)
	require.NoError(t, err)
	assert.Nil(t, answers)
}

func TestRunProgram(t *testing.T) {
	var in, out bytes.Buffer
	_, _ = in.WriteString("Acme\rkeys/a.p12\ry")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	answers, err := run(ctx, []Question{
		{Prompt: "Publisher name"},
		{Prompt: "PKCS#12 path"},
		{Prompt: "Replace?", Confirm: true},
	}, &in, &out)
	require.NoError(t, err)
	assert.Equal(t, []Answer{
		{Text: "Acme"},
		{Text: "keys/a.p12"},
		{Text: "y", Yes: true},
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
