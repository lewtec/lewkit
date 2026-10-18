package term

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeskClicksThroughTheRoom(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := model.View()
	assert.Equal(t, tea.MouseModeCellMotion, view.MouseMode)
	assert.Equal(t, "lewkit term", view.WindowTitle)
	assert.True(t, model.palette)

	clickRail(t, model, "lew")
	assert.False(t, model.palette)
	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "the desk is clear")
	assert.Contains(t, plain, "quiet, for now")
	assert.NotContains(t, plain, "taskbar")

	clickRail(t, model, "lew")
	assert.True(t, model.palette)
	_, lines := renderPlain(model)
	_, cmd := model.Update(tea.MouseClickMsg{X: 1, Y: len(lines) - 1, Button: tea.MouseRight})
	assert.Nil(t, cmd)
	assert.True(t, model.palette)

	clickRail(t, model, "lew")
	model.order = []string{"ed", "docs", "n"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true, Rows: 10, Cols: 40}}
	model.byID["docs"] = Pane{PaneInfo: PaneInfo{ID: "docs", Path: "work/docs", Title: "md", Alive: true, Rows: 10, Cols: 40}}
	model.byID["n"] = Pane{PaneInfo: PaneInfo{ID: "n", Path: "notes", Title: "notes", Alive: true, Rows: 10, Cols: 40}}
	clickRail(t, model, "drawer")
	assert.True(t, model.drawer)
	assert.False(t, model.palette)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "the drawer")
	assert.Contains(t, plain, "sessions live here")
	assert.NotContains(t, plain, "├ editor")

	runCmd(clickLine(t, model, "work", "work"))
	assert.True(t, model.drawer)
	assert.Equal(t, "work", model.shown)
	assert.Equal(t, "work", model.followPrefix)
	assert.False(t, model.followAll)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "├ editor")
	assert.Contains(t, plain, "└ docs")

	runCmd(clickLine(t, model, "editor", "editor"))
	assert.False(t, model.drawer)
	assert.Equal(t, "ed", model.focus)
	assert.Equal(t, "work", model.followPrefix)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "work/editor")
	assert.NotContains(t, plain, "the drawer")

	clickRail(t, model, "drawer")
	assert.True(t, model.drawer)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "├ editor")

	runCmd(clickClose(t, model, "notes"))
	assert.Equal(t, []string{"n"}, api.removes)
	plain, _ = renderPlain(model)
	assert.NotContains(t, plain, "notes")
	assert.Contains(t, plain, "├ editor")

	runCmd(clickClose(t, model, "work"))
	assert.Equal(t, []string{"n", "ed", "docs"}, api.removes)
	assert.True(t, model.followAll)
	assert.Empty(t, model.shown)
	plain, _ = renderPlain(model)
	assert.NotContains(t, plain, "work")
	assert.Contains(t, plain, "nothing in here yet")
}

func TestDeskTabsAndPaletteChoices(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model.palette = false
	model.order = []string{"a", "b"}
	model.byID["a"] = Pane{PaneInfo: PaneInfo{ID: "a", Path: "alpha", Title: "alpha", Alive: true, Rows: 10, Cols: 40}}
	model.byID["b"] = Pane{PaneInfo: PaneInfo{ID: "b", Path: "beta", Title: "beta", Alive: true, Rows: 10, Cols: 40}}
	model.focus = "a"
	model.touch("a")
	model.touch("b")

	runCmd(clickRail(t, model, "alpha"))
	assert.Equal(t, "a", model.focus)
	assert.True(t, model.followAll)
	assert.False(t, model.drawer)
	assert.Empty(t, api.watches)

	model.shown = "beta"
	model.followAll = false
	model.followPrefix = ""
	model.follow = []string{"b"}
	model.focus = "b"
	runCmd(clickRail(t, model, "alpha"))
	assert.Equal(t, "a", model.focus)
	assert.Equal(t, "alpha", model.shown)
	assert.False(t, model.followAll)
	assert.Equal(t, []string{"a"}, model.follow)
	assert.False(t, model.drawer)

	model.palette = true
	model.cursor = 0
	model.query = ""
	model.step = StepActions
	model.followAll = true
	model.follow = nil
	model.shown = ""
	model.order = nil
	runCmd(clickLine(t, model, "New pane", "New pane"))
	require.Len(t, api.creates, 1)
	assert.Equal(t, 100, api.creates[0].Cols)
	assert.Equal(t, 29, api.creates[0].Rows)

	model.cursor = 10
	model.step = StepActions
	model.palette = true
	runCmd(clickLine(t, model, "Open folder", "Open folder"))
	assert.Equal(t, StepLine, model.step)
	assert.Equal(t, ActionFolder, model.action)
}

func TestDrawerNestsASession(t *testing.T) {
	model := newModel(t.Context(), &fakeAPI{}, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.shown = "work"
	model.order = []string{"main", "docs"}
	model.byID["main"] = Pane{PaneInfo: PaneInfo{ID: "main", Path: "work/src/main", Title: "go", Alive: true}}
	model.byID["docs"] = Pane{PaneInfo: PaneInfo{ID: "docs", Path: "work/docs", Title: "md", Alive: true}}

	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "├ src")
	assert.Contains(t, plain, "│ └ main")
	assert.Contains(t, plain, "└ docs")
	assert.NotContains(t, plain, "taskbar")
}

func TestNoticePaneClosedDropsThePane(t *testing.T) {
	model := newModel(t.Context(), &fakeAPI{}, "")
	model.order = []string{"n", "ed"}
	model.byID["n"] = Pane{PaneInfo: PaneInfo{ID: "n", Path: "notes", Alive: true}}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Alive: true}}
	model.focus = "n"
	model.recent = []string{"n", "ed"}
	model.follow = []string{"n", "ed"}
	model.Update(Event{Notice: NoticePaneClosed, Pane: &Pane{PaneInfo: PaneInfo{ID: "n"}}})
	assert.Equal(t, []string{"ed"}, model.order)
	assert.Equal(t, []string{"ed"}, model.recent)
	assert.Equal(t, []string{"ed"}, model.follow)
	assert.Empty(t, model.focus)
	_, ok := model.byID["n"]
	assert.False(t, ok)
}

func TestDeskDrawerKeysStayOffThePane(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.order = []string{"ed", "n"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true}}
	model.byID["n"] = Pane{PaneInfo: PaneInfo{ID: "n", Path: "notes", Title: "notes", Alive: true}}
	model.focus = "ed"

	_, cmd := model.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	assert.Nil(t, cmd)
	assert.Empty(t, api.writes)

	_, cmd = model.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.Nil(t, cmd)
	assert.Empty(t, api.writes)
	assert.True(t, model.drawer)

	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.Equal(t, "notes", model.shown)
	assert.Equal(t, []string{"n"}, model.follow)
	assert.True(t, model.drawer)

	model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.False(t, model.drawer)

	_, cmd = model.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	require.NotNil(t, cmd)
	_ = cmd()
	require.Len(t, api.writes, 1)
	assert.Equal(t, "a", api.writes[0].Data)
}

func renderPlain(m *Model) (string, []string) {
	content, _, _, _ := m.render()
	plain := stripANSI(content)
	return plain, strings.Split(plain, "\n")
}

func clickRail(t *testing.T, m *Model, word string) tea.Cmd {
	t.Helper()
	_, lines := renderPlain(m)
	y := len(lines) - 1
	at := strings.Index(lines[y], word)
	require.GreaterOrEqual(t, at, 0, "rail %q has no %q", lines[y], word)
	_, cmd := m.Update(tea.MouseClickMsg{X: colAt(lines[y], at), Y: y, Button: tea.MouseLeft})
	return cmd
}

func clickLine(t *testing.T, m *Model, row, word string) tea.Cmd {
	t.Helper()
	_, lines := renderPlain(m)
	for y, text := range lines {
		if !strings.Contains(text, row) {
			continue
		}
		at := strings.Index(text, word)
		if at < 0 {
			continue
		}
		_, cmd := m.Update(tea.MouseClickMsg{X: colAt(text, at), Y: y, Button: tea.MouseLeft})
		return cmd
	}
	require.Fail(t, "missing click target", "no %q on a line containing %q", word, row)
	return nil
}

func clickClose(t *testing.T, m *Model, key string) tea.Cmd {
	t.Helper()
	_, lines := renderPlain(m)
	for y, text := range lines {
		if !strings.Contains(text, key) || !strings.HasSuffix(text, "x") {
			continue
		}
		at := strings.LastIndex(text, "x")
		_, cmd := m.Update(tea.MouseClickMsg{X: colAt(text, at), Y: y, Button: tea.MouseLeft})
		return cmd
	}
	require.Fail(t, "missing close mark", key)
	return nil
}

func runCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return
	}
	for _, inner := range batch {
		runCmd(inner)
	}
}

func colAt(text string, byteAt int) int {
	return runewidth.StringWidth(text[:byteAt])
}

func stripANSI(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
