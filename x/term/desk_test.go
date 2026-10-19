package term

import (
	"fmt"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
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
	assert.False(t, model.palette)
	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "the desk is clear")
	assert.Contains(t, plain, "quiet, for now")
	assert.NotContains(t, plain, "New pane")
	assert.NotContains(t, plain, "taskbar")

	clickRail(t, model, "lew")
	assert.True(t, model.palette)
	_, lines := renderPlain(model)
	_, cmd := model.Update(tea.MouseClickMsg{X: 1, Y: len(lines) - 1, Button: tea.MouseRight})
	assert.Nil(t, cmd)
	assert.True(t, model.palette)

	clickRail(t, model, "lew")
	assert.False(t, model.palette)
	model.order = []string{"ed", "docs", "n"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true, Rows: 10, Cols: 40}}
	model.byID["docs"] = Pane{PaneInfo: PaneInfo{ID: "docs", Path: "work/docs", Title: "md", Alive: true, Rows: 10, Cols: 40}}
	model.byID["n"] = Pane{PaneInfo: PaneInfo{ID: "n", Path: "notes", Title: "notes", Alive: true, Rows: 10, Cols: 40}}
	clickRail(t, model, "drawer")
	assert.True(t, model.drawer)
	assert.False(t, model.palette)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "the drawer")
	assert.Contains(t, plain, "* /")
	assert.Contains(t, plain, " · work  "+roomLabel(model.cwd))
	assert.Contains(t, plain, "looking around")
	assert.Contains(t, plain, "sessions live here")
	assert.NotContains(t, plain, "standing in ")
	assert.Contains(t, plain, "├ editor")

	runCmd(clickLine(t, model, " · work", "work"))
	assert.True(t, model.drawer)
	assert.Equal(t, "work", model.shown)
	assert.Equal(t, "work", model.followPrefix)
	assert.False(t, model.followAll)
	assert.Empty(t, model.heldSession)
	assert.Empty(t, model.held)
	assert.Equal(t, "ed", model.armed)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "├ editor")
	assert.Contains(t, plain, "└ docs")

	runCmd(clickLine(t, model, "├ editor", "editor"))
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
	assert.NotContains(t, plain, " · work")
	assert.NotContains(t, plain, " * work")
	assert.NotContains(t, plain, "├ editor")
	assert.Contains(t, plain, "nothing in here yet")
	assert.NotContains(t, plain, "standing in ")
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
	assert.Equal(t, "a", model.find)
	assert.True(t, model.drawer)

	_, cmd = model.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.Nil(t, cmd)
	assert.Empty(t, api.writes)
	assert.True(t, model.drawer)

	model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.Empty(t, model.find)
	assert.True(t, model.drawer)

	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
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

func TestDrawerSearchFindsPanesAndActions(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model.palette = false
	model.drawer = true
	model.order = []string{"ed", "n"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true}}
	model.byID["n"] = Pane{PaneInfo: PaneInfo{ID: "n", Path: "notes", Title: "notes", Alive: true}}
	model.focus = "ed"

	plain, _ := renderPlain(model)
	assert.NotContains(t, plain, "find a pane")
	assert.Contains(t, plain, "+ directory")
	assert.Contains(t, plain, "the drawer")
	assert.NotContains(t, plain, "taskbar")

	model.drawer = false
	term, _ := renderPlain(model)
	assert.NotContains(t, term, "the drawer")
	assert.NotContains(t, term, "find a pane")
	model.drawer = true

	dir := "/lewkit-room"
	model.cwd = "/client"
	model.gitBy = map[string]string{dir: "main · tidy"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true, Dir: dir}}
	model.byID["n"] = Pane{PaneInfo: PaneInfo{ID: "n", Path: "notes", Title: "notes", Alive: true, Dir: "/notes-room"}}

	for _, r := range "lewkit-room" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, " · work")
	assert.Contains(t, plain, "lewkit-room")
	assert.NotContains(t, plain, " · notes")
	assert.NotContains(t, plain, "+ directory")
	assert.NotContains(t, plain, "New pane")

	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.True(t, model.drawer)
	assert.Equal(t, "work", model.shown)
	assert.Equal(t, "ed", model.armed)
	assert.Equal(t, "lewkit-room", model.find)

	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.False(t, model.drawer)
	assert.Equal(t, "ed", model.focus)
	assert.Empty(t, model.find)

	model.drawer = true
	model.armed = ""
	for _, r := range "hx" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "work/editor")
	assert.NotContains(t, plain, " · work")
	assert.NotContains(t, plain, " · notes")

	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.True(t, model.drawer)
	assert.Equal(t, "ed", model.focus)
	assert.Equal(t, "ed", model.armed)
	assert.Equal(t, "hx", model.find)

	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.False(t, model.drawer)
	assert.Equal(t, "ed", model.focus)
	assert.Equal(t, "work", model.shown)
	assert.Empty(t, model.find)
	assert.Empty(t, api.creates)

	_, cmd = model.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	runCmd(cmd)
	assert.True(t, model.drawer)
	assert.Empty(t, api.writes)

	for _, r := range "uber" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "Attach ubersession")
	assert.NotContains(t, plain, "New pane")
	assert.NotContains(t, plain, "Follow pane")
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.True(t, model.followAll)
	assert.False(t, model.drawer)
	assert.Empty(t, api.creates)

	model.drawer = true
	model.palette = false
	for _, r := range "editor" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	cmd = clickLine(t, model, "work/editor", "editor")
	assert.Nil(t, cmd)
	assert.True(t, model.drawer)
	assert.Equal(t, "ed", model.armed)
	assert.Empty(t, model.held)
	cmd = clickLine(t, model, "work/editor", "editor")
	runCmd(cmd)
	assert.False(t, model.drawer)
	assert.Equal(t, "ed", model.focus)
}

func TestDrawerSearchOpensAPaneAndAFolder(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model.palette = false
	model.drawer = true
	model.cwd = "/lewkit-room"

	for _, r := range "new pane" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "New pane")
	assert.NotContains(t, plain, "New command")
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	_ = cmd()
	require.Len(t, api.creates, 1)
	assert.Equal(t, 38, api.creates[0].Rows)
	assert.Equal(t, 100, api.creates[0].Cols)
	assert.Equal(t, "/lewkit-room", api.creates[0].Dir)
	assert.False(t, model.drawer)

	model.palette = false
	model.drawer = true
	for _, r := range "open folder" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "Open folder")
	assert.NotContains(t, plain, "New pane")
	assert.NotContains(t, plain, "Follow folder")
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.False(t, model.drawer)
	assert.True(t, model.palette)
	assert.Equal(t, StepLine, model.step)
	assert.Equal(t, ActionFolder, model.action)
}

func TestModeSwitchDoesNotRunAnAction(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.order = []string{"ed"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true}}
	model.focus = "ed"
	model.follow = []string{"ed"}

	_, cmd := model.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	runCmd(cmd)
	assert.True(t, model.drawer)
	assert.False(t, model.palette)
	assert.Empty(t, api.writes)
	assert.Empty(t, api.creates)

	model.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModSuper})
	assert.True(t, model.drawer)
	assert.Empty(t, api.writes)

	model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.False(t, model.drawer)

	model.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModMeta})
	assert.True(t, model.drawer)
	assert.Empty(t, api.writes)

	model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	model.palette = true
	model.Update(tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl})
	assert.False(t, model.palette)
	assert.True(t, model.drawer)
	assert.Empty(t, api.creates)
}

func TestEnterOnAPaneThenAFolderDoesNotNest(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.shown = "work"
	model.made = []string{"pockets"}
	model.order = []string{"ed", "main"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true}}
	model.byID["main"] = Pane{PaneInfo: PaneInfo{ID: "main", Path: "work/src/main", Title: "go", Alive: true}}

	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.True(t, model.drawer)
	assert.Equal(t, "ed", model.armed)
	assert.Empty(t, model.held)
	assert.Empty(t, api.moves)
	assert.Equal(t, "work/editor", model.byID["ed"].Path)

	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.True(t, model.drawer)
	assert.Empty(t, model.held)
	assert.Empty(t, model.place)
	assert.Empty(t, api.moves)
	assert.Equal(t, "work/editor", model.byID["ed"].Path)
	assert.Equal(t, "work/src/main", model.byID["main"].Path)
	plain, _ := renderPlain(model)
	assert.NotContains(t, plain, "pocket work/src")
	assert.Contains(t, plain, "├ editor")
	assert.Contains(t, plain, "└ src")

	model.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	model.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	model.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Equal(t, "pockets", model.shown)
	assert.Empty(t, api.moves)
	assert.Equal(t, "work/editor", model.byID["ed"].Path)
	plain, _ = renderPlain(model)
	assert.NotContains(t, plain, "pockets/work/editor")
	assert.Contains(t, plain, "· work")

	model.shown = "work"
	model.drawer = true
	model.armed = ""
	model.held = ""
	model.focus = "ed"
	model.drawerAt = 3
	model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	assert.Equal(t, "ed", model.held)
	assert.Empty(t, api.moves)

	model.drawerAt = 2
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	applyCmd(model, cmd)
	assert.Empty(t, api.moves)
	assert.Equal(t, "work/editor", model.byID["ed"].Path)
	assert.Empty(t, model.held)
	assert.Empty(t, model.byID["ed"].Dir)

	model.drawerAt = 3
	model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	assert.Equal(t, "ed", model.held)
	model.drawerAt = 4
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	applyCmd(model, cmd)
	require.Equal(t, []movedMsg{{id: "ed", path: "work/src/editor"}}, api.moves)
	assert.Equal(t, "work/src/editor", model.byID["ed"].Path)
	assert.Empty(t, model.byID["ed"].Dir)
	assert.Empty(t, model.held)
}

func TestTwoEntersStayOnThePaneYouArmed(t *testing.T) {
	model := newModel(t.Context(), &fakeAPI{}, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.shown = "work"
	model.order = []string{"ed", "docs"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true}}
	model.byID["docs"] = Pane{PaneInfo: PaneInfo{ID: "docs", Path: "work/docs", Title: "md", Alive: true}}

	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.True(t, model.drawer)
	assert.Equal(t, "ed", model.armed)
	assert.Equal(t, "ed", model.focus)

	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.True(t, model.drawer)
	assert.Equal(t, "docs", model.armed)
	assert.Equal(t, "docs", model.focus)

	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.False(t, model.drawer)
	assert.Equal(t, "docs", model.focus)
	assert.Equal(t, "work", model.shown)
}

func TestDoubleEnterSelectsThePaneInTheList(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.False(t, model.palette)
	model.drawer = true
	model.palette = true
	model.order = []string{"ed", "docs", "n"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true}}
	model.byID["docs"] = Pane{PaneInfo: PaneInfo{ID: "docs", Path: "work/docs", Title: "md", Alive: true}}
	model.byID["n"] = Pane{PaneInfo: PaneInfo{ID: "n", Path: "notes", Title: "notes", Alive: true}}

	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.True(t, model.drawer)
	assert.Equal(t, "work", model.shown)
	assert.Equal(t, "ed", model.armed)
	assert.Empty(t, api.creates)

	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.False(t, model.drawer)
	assert.Equal(t, "ed", model.focus)
	assert.Equal(t, "work", model.shown)
	assert.Empty(t, api.creates)

	model.drawer = true
	model.palette = true
	model.shown = "work"
	model.armed = ""
	model.drawerAt = 0
	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.True(t, model.drawer)
	assert.Equal(t, "ed", model.armed)

	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.False(t, model.drawer)
	assert.Equal(t, "ed", model.focus)
	assert.Empty(t, api.creates)

	model.drawer = true
	model.palette = true
	model.shown = ""
	model.armed = ""
	model.drawerAt = 0
	model.focus = ""
	model.order = []string{"n"}
	model.byID = map[string]Pane{
		"n": {PaneInfo: PaneInfo{ID: "n", Path: "notes", Title: "notes", Alive: true}},
	}
	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.True(t, model.drawer)
	assert.Equal(t, "n", model.armed)
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	runCmd(cmd)
	assert.False(t, model.drawer)
	assert.Equal(t, "n", model.focus)
	assert.Empty(t, api.creates)
}

func TestPickPaintsWithLipgloss(t *testing.T) {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.ANSI)
	painted := stylePick.Renderer(r).Render(" lew ")
	assert.NotEqual(t, " lew ", painted)
	assert.Equal(t, " lew ", stripANSI(painted))
}

func TestDrawerShowsTheRoomAndMovesAPane(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	dir := "/lewkit-room"
	model.cwd = dir
	model.gitBy = map[string]string{dir: "main · tidy"}
	model.order = []string{"ed", "docs"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true, Dir: dir}}
	model.byID["docs"] = Pane{PaneInfo: PaneInfo{ID: "docs", Path: "work/docs", Title: "md", Alive: true, Dir: dir}}
	model.focus = "ed"

	plain, _ := renderPlain(model)
	assert.Contains(t, plain, " · work  /lewkit-room  main · tidy")
	assert.NotContains(t, plain, "standing in ")
	assert.Contains(t, plain, "+ directory")
	assert.Contains(t, plain, "├ editor")

	runCmd(clickLine(t, model, "+ directory", "+ directory"))
	assert.True(t, model.naming)
	assert.True(t, model.drawer)
	model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.False(t, model.naming)
	assert.True(t, model.drawer)
	assert.Empty(t, api.writes)

	runCmd(clickLine(t, model, "+ directory", "+ directory"))
	_, cmd := model.Update(tea.KeyPressMsg{Text: "p", Code: 'p'})
	assert.Nil(t, cmd)
	assert.Empty(t, api.writes)
	_, cmd = model.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	assert.Nil(t, cmd)
	assert.Equal(t, "p", model.name)
	assert.True(t, model.naming)
	assert.True(t, model.drawer)
	for _, r := range "ockets" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	assert.IsType(t, wroteMsg{}, cmd())
	assert.Equal(t, []string{"pockets"}, api.dirCreates)
	assert.Equal(t, []string{"pockets"}, model.made)
	assert.Equal(t, "pockets", model.place)
	assert.Equal(t, "pockets", model.shown)
	assert.Empty(t, api.moves)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "pocket pockets")
	assert.Contains(t, plain, " * pockets")

	runCmd(clickLine(t, model, "+ directory", "+"))
	for _, r := range "inner" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	assert.IsType(t, wroteMsg{}, cmd())
	assert.Equal(t, []string{"pockets", "pockets/inner"}, api.dirCreates)
	assert.Equal(t, []string{"pockets", "pockets/inner"}, model.made)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "└ inner")

	cmd = clickSessionCarry(t, model, " · work")
	assert.Nil(t, cmd)
	assert.Equal(t, "work", model.heldSession)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "carrying work")

	cmd = clickLine(t, model, " · pockets", "pockets")
	applyCmd(model, cmd)
	require.Equal(t, []movedMsg{
		{id: "ed", path: "pockets/work/editor"},
		{id: "docs", path: "pockets/work/docs"},
	}, api.moves)
	assert.Equal(t, "pockets/work/editor", model.byID["ed"].Path)
	assert.Equal(t, "pockets/work/docs", model.byID["docs"].Path)
	assert.Equal(t, dir, model.byID["ed"].Dir)
	assert.Equal(t, "pockets/work", model.shown)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "work  /lewkit-room  main · tidy")
	assert.Contains(t, plain, "├ editor")
	assert.Contains(t, plain, "└ docs")
	assert.Contains(t, plain, "├ inner")
	assert.NotContains(t, plain, "work/editor")
}

func TestSessionRowKeepsTheGitStatus(t *testing.T) {
	model := newModel(t.Context(), &fakeAPI{}, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	dir := "/tmp/" + strings.Repeat("directory", 12)
	model.gitBy = map[string]string{dir: "main · tidy"}
	model.order = []string{"ed"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true, Dir: dir}}
	model.focus = "ed"

	plain, _ := renderPlain(model)
	assert.Contains(t, plain, " · work")
	assert.Contains(t, plain, "main · tidy")
	assert.Contains(t, plain, "→ x")
}

func TestDeskShowsTheFocusedPane(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model.palette = false
	model.followAll = true
	model.order = []string{"a", "b"}
	model.byID["a"] = Pane{PaneInfo: PaneInfo{ID: "a", Path: "alpha", Title: "alpha", Alive: true, Rows: 10, Cols: 20}}
	model.byID["b"] = Pane{PaneInfo: PaneInfo{ID: "b", Path: "beta", Title: "beta", Alive: true, Rows: 10, Cols: 20}}
	model.focus = "a"

	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "alpha")
	assert.NotContains(t, plain, "beta")

	_, cmd := model.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	require.NotNil(t, cmd)
	_ = cmd()
	require.Len(t, api.writes, 1)
	assert.Equal(t, 100, api.writes[0].Cols)
	assert.Equal(t, 38, api.writes[0].Rows)
}

func TestDrawerLeavesFoldersOpen(t *testing.T) {
	model := newModel(t.Context(), &fakeAPI{}, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.made = []string{"archive", "archive/bin"}
	model.order = []string{"ed", "page"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true}}
	model.byID["page"] = Pane{PaneInfo: PaneInfo{ID: "page", Path: "notes/page", Title: "md", Alive: true}}

	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "└ editor")
	assert.Contains(t, plain, "└ page")
	assert.Contains(t, plain, "└ bin")
	assert.Contains(t, plain, " · work")
	assert.Contains(t, plain, " · notes")

	model.shown = "work"
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, " * work")
	assert.Contains(t, plain, "└ page")
	assert.Contains(t, plain, "└ bin")

	for _, r := range "page" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "notes/page")
	assert.NotContains(t, plain, "└ editor")
	assert.NotContains(t, plain, "└ bin")
	assert.NotContains(t, plain, "+ directory")
}

func TestServerLevelWarns(t *testing.T) {
	model := newModel(t.Context(), &fakeAPI{}, "")
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model.made = []string{"local"}
	model.facts = map[string]map[string]string{"local": {"task": "stay"}}

	model.Update(loadedMsg{info: SessionInfo{Level: 0}})
	assert.Equal(t, []string{"local"}, model.made)
	assert.Equal(t, "stay", model.facts["local"]["task"])
	plain, _ := renderPlain(model)
	assert.Contains(t, plain, fmt.Sprintf("server protocol 1, client protocol %d", ProtocolLevel))
	assert.Contains(t, plain, "restart the server")

	model.drawer = true
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, fmt.Sprintf("server protocol 1, client protocol %d", ProtocolLevel))

	model.Update(loadedMsg{info: SessionInfo{Level: 2, Dirs: []string{"kept"}}})
	assert.Equal(t, []string{"kept"}, model.made)
	assert.Equal(t, "stay", model.facts["local"]["task"])
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, fmt.Sprintf("server protocol 2, client protocol %d", ProtocolLevel))
	assert.Contains(t, plain, "restart the server")

	model.Update(loadedMsg{info: SessionInfo{
		Level: ProtocolLevel,
		Dirs:  []string{"pockets"},
		Facts: []Fact{{Path: "pockets", Key: "task", Value: "parser"}},
	}})
	assert.Empty(t, model.levelText)
	assert.Equal(t, []string{"pockets"}, model.made)
	assert.Equal(t, "parser", model.facts["pockets"]["task"])
	plain, _ = renderPlain(model)
	assert.NotContains(t, plain, "server protocol")
	assert.Contains(t, plain, "pockets")

	model.Update(loadedMsg{info: SessionInfo{Level: ProtocolLevel + 1, Dirs: []string{"pockets"}}})
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, fmt.Sprintf("server protocol %d, client protocol %d", ProtocolLevel+1, ProtocolLevel))
	assert.NotContains(t, plain, "restart the server")
	assert.Equal(t, []string{"pockets"}, model.made)
}

func TestDrawerUsesReportedCwdAndFacts(t *testing.T) {
	model := newModel(t.Context(), &fakeAPI{}, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.shown = "work"
	model.focus = "ed"
	model.order = []string{"ed"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{
		ID: "ed", Path: "work/editor", Title: "hx", Alive: true,
		Dir: "/old", Cwd: "/new",
	}}
	long := strings.Repeat("x", 50)
	model.facts = map[string]map[string]string{"work": {
		"task":     "parser",
		"progress": "halfway",
		"state":    long,
		"cwd":      "/new",
	}}

	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "/new")
	assert.NotContains(t, plain, "/old")
	assert.Contains(t, plain, "task parser")
	assert.Contains(t, plain, "progress halfway")
	assert.Contains(t, plain, "state "+strings.Repeat("x", 40))
	assert.NotContains(t, plain, strings.Repeat("x", 41))
	assert.NotContains(t, plain, "cwd /new")

	changed := model.applyEvent(Event{Notice: NoticePane, Pane: &Pane{PaneInfo: PaneInfo{
		ID: "ed", Path: "work/editor", Dir: "/old", Cwd: "/newer",
	}}})
	assert.True(t, changed)
	assert.Equal(t, "/newer", paneRoom(model.byID["ed"]))
	changed = model.applyEvent(Event{Notice: NoticePane, Pane: &Pane{PaneInfo: PaneInfo{
		ID: "ed", Path: "work/editor", Dir: "/old", Cwd: "/newer",
	}}})
	assert.False(t, changed)
}

func TestFactsFollowThePaneID(t *testing.T) {
	model := newModel(t.Context(), &fakeAPI{}, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.focus = "ed"
	model.order = []string{"ed", "other"}
	model.made = []string{"pockets"}
	model.shown = "pockets/work"
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{
		ID: "ed", Path: "pockets/work/editor", Title: "hx", Alive: true,
		Dir: "/old", Cwd: "/new",
	}}
	model.byID["other"] = Pane{PaneInfo: PaneInfo{
		ID: "other", Path: "pockets/work/other", Title: "vi", Alive: true,
	}}
	model.facts = map[string]map[string]string{
		"ed":      {"task": "local"},
		"other":   {"task": "sibling"},
		"pockets": {"state": "shared", "progress": "parent"},
	}

	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "task local")
	assert.Contains(t, plain, "state shared")
	assert.Contains(t, plain, "progress parent")
	assert.NotContains(t, plain, "task sibling")

	model.focus = "other"
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "task sibling")
	assert.Contains(t, plain, "state shared")
	assert.NotContains(t, plain, "task local")

	model.focus = "ed"
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{
		ID: "ed", Path: "archive/editor", Title: "hx", Alive: true,
		Dir: "/old", Cwd: "/new",
	}}
	model.shown = "archive"
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "task local")
	assert.NotContains(t, plain, "state shared")
	assert.NotContains(t, plain, "progress parent")
}

func TestDrawerForgetsAnEmptyDirectory(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.made = []string{"spare", "spare/bin"}
	model.shown = "spare"
	model.place = "spare/bin"
	model.followAll = false
	model.followPrefix = "spare"

	plain, _ := renderPlain(model)
	assert.Contains(t, plain, " * spare")
	assert.Contains(t, plain, "└ bin")

	cmd := clickClose(t, model, "spare")
	require.NotNil(t, cmd)
	assert.IsType(t, wroteMsg{}, cmd())
	assert.Equal(t, []string{"spare"}, api.dirRemoves)
	assert.Empty(t, model.made)
	assert.Empty(t, model.place)
	assert.Empty(t, model.shown)
	assert.True(t, model.followAll)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, "nothing in here yet")
	assert.NotContains(t, plain, "spare")
}

func renderPlain(m *Model) (string, []string) {
	content, _, _, _ := m.render()
	plain := stripANSI(content)
	return plain, strings.Split(plain, "\n")
}

func TestDrawerWheelScrollsLikeTheKeys(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.cwd = "/room"
	for i := range 30 {
		id := fmt.Sprintf("s%02d", i)
		model.order = append(model.order, id)
		model.byID[id] = Pane{PaneInfo: PaneInfo{ID: id, Path: id, Title: id, Alive: true, Dir: "/room"}}
	}
	model.gitBy = map[string]string{"/room": "main · tidy"}

	model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	assert.Equal(t, 1, model.drawerAt)
	model.drawerAt = 0
	_, cmd := model.Update(tea.MouseWheelMsg{X: 1, Y: 1, Button: tea.MouseWheelDown})
	assert.Nil(t, cmd)
	assert.Equal(t, 1, model.drawerAt)
	assert.Empty(t, api.writes)

	for range 40 {
		_, cmd = model.Update(tea.MouseWheelMsg{X: 1, Y: 1, Button: tea.MouseWheelDown})
		assert.Nil(t, cmd)
	}
	assert.Equal(t, 30, model.drawerAt)
	plain, _ := renderPlain(model)
	assert.NotContains(t, plain, " · s00")
	assert.Contains(t, plain, " · s29")
	assert.True(t, rootShown(plain))
	assert.Empty(t, api.writes)

	for range 40 {
		_, cmd = model.Update(tea.MouseWheelMsg{X: 1, Y: 1, Button: tea.MouseWheelUp})
		assert.Nil(t, cmd)
	}
	assert.Equal(t, -1, model.drawerAt)
	plain, _ = renderPlain(model)
	assert.Contains(t, plain, " · s00")
	assert.True(t, rootShown(plain))

	at := model.drawerAt
	_, cmd = model.Update(tea.MouseWheelMsg{X: 1, Y: 1, Button: tea.MouseWheelLeft})
	assert.Nil(t, cmd)
	assert.Equal(t, at, model.drawerAt)

	model.drawer = false
	model.palette = true
	model.drawerAt = 4
	_, cmd = model.Update(tea.MouseWheelMsg{X: 1, Y: 1, Button: tea.MouseWheelDown})
	assert.Nil(t, cmd)
	assert.Equal(t, 4, model.drawerAt)
	assert.Empty(t, api.writes)
}

func TestDrawerClickSelectsASession(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.order = []string{"ed", "n"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true}}
	model.byID["n"] = Pane{PaneInfo: PaneInfo{ID: "n", Path: "notes", Title: "notes", Alive: true}}

	runCmd(clickLine(t, model, " · work", "work"))
	assert.True(t, model.drawer)
	assert.Equal(t, "work", model.shown)
	assert.Equal(t, "work", model.followPrefix)
	assert.False(t, model.followAll)
	assert.Empty(t, model.heldSession)
	assert.Equal(t, "ed", model.armed)
	assert.Empty(t, api.writes)

	runCmd(clickLine(t, model, "* work", "work"))
	assert.False(t, model.drawer)
	assert.Equal(t, "ed", model.focus)
	assert.Equal(t, "work", model.shown)
	assert.Empty(t, model.heldSession)
	assert.Empty(t, api.writes)
}

func TestDrawerClickNestsAPaneThenAFolder(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.shown = "work"
	model.made = []string{"pockets"}
	model.order = []string{"ed", "main"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true, Dir: "/lewkit-room"}}
	model.byID["main"] = Pane{PaneInfo: PaneInfo{ID: "main", Path: "work/src/main", Title: "go", Alive: true, Dir: "/lewkit-room"}}

	model.drawerAt = 3
	model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	assert.True(t, model.drawer)
	assert.Equal(t, "ed", model.held)
	assert.Empty(t, api.moves)
	assert.Equal(t, "work/editor", model.byID["ed"].Path)

	cmd := clickLine(t, model, " · pockets", "pockets")
	applyCmd(model, cmd)
	require.Equal(t, []movedMsg{{id: "ed", path: "pockets/editor"}}, api.moves)
	assert.Equal(t, "pockets/editor", model.byID["ed"].Path)
	assert.Equal(t, "/lewkit-room", model.byID["ed"].Dir)
	assert.Equal(t, "pockets", model.place)
	assert.True(t, model.drawer)
	assert.Empty(t, model.held)
}

func TestDrawerClickNestsAFolderAndUnnestsAtRoot(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.made = []string{"archive", "pockets"}
	model.order = []string{"note"}
	model.byID["note"] = Pane{PaneInfo: PaneInfo{ID: "note", Path: "pockets/note", Title: "n", Alive: true, Dir: "/lewkit-room"}}

	model.drawerAt = 2
	model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	assert.Equal(t, "pockets", model.heldFolder)
	assert.Empty(t, api.moves)
	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "carrying pockets")

	model.drawerAt = 1
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	applyCmd(model, cmd)
	assert.Equal(t, []string{"archive", "archive/pockets"}, model.made)
	assert.Equal(t, [][2]string{{"pockets", "archive/pockets"}}, api.dirMoves)
	assert.Equal(t, "archive/pockets/note", model.byID["note"].Path)
	assert.Equal(t, "/lewkit-room", model.byID["note"].Dir)
	assert.Equal(t, "archive", model.place)
	assert.Empty(t, model.heldFolder)

	model.drawerAt = 1
	model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	assert.Equal(t, "archive", model.heldFolder)
	model.drawerAt = 2
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Equal(t, ErrPath.Error(), model.errText)
	assert.Equal(t, "archive", model.heldFolder)
	assert.Equal(t, "archive/pockets/note", model.byID["note"].Path)

	model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.Empty(t, model.heldFolder)
	assert.True(t, model.drawer)
	assert.Equal(t, "archive/pockets/note", model.byID["note"].Path)

	model.drawerAt = 2
	model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	assert.Equal(t, "archive/pockets", model.heldFolder)
	model.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	model.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	model.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	assert.Equal(t, -1, model.drawerAt)
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	applyCmd(model, cmd)
	assert.Equal(t, []string{"archive", "pockets"}, model.made)
	assert.Equal(t, [][2]string{{"pockets", "archive/pockets"}, {"archive/pockets", "pockets"}}, api.dirMoves)
	assert.Equal(t, "pockets/note", model.byID["note"].Path)
	assert.Equal(t, "/lewkit-room", model.byID["note"].Dir)
	assert.Empty(t, model.place)
	assert.Empty(t, model.heldFolder)
	assert.True(t, model.drawer)

	before := len(api.moves)
	model.drawerAt = 2
	model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	assert.Equal(t, "pockets", model.heldFolder)
	cmd = clickRoot(t, model)
	assert.Nil(t, cmd)
	assert.Empty(t, model.heldFolder)
	assert.Equal(t, "pockets/note", model.byID["note"].Path)
	assert.Len(t, api.moves, before)

	model.drawerAt = 2
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Equal(t, "pockets", model.shown)
	assert.Empty(t, model.heldFolder)
	assert.Len(t, api.moves, before)

	model.find = "arc"
	model.heldFolder = "archive"
	model.drawer = true
	model.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	assert.Empty(t, model.heldFolder)
	assert.Equal(t, "arc", model.find)
	assert.True(t, model.drawer)
}

func TestDrawerRootUnnestsASession(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.shown = "pockets"
	model.made = []string{"pockets"}
	model.order = []string{"ed", "docs"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "pockets/work/editor", Title: "hx", Alive: true, Dir: "/lewkit-room"}}
	model.byID["docs"] = Pane{PaneInfo: PaneInfo{ID: "docs", Path: "pockets/work/docs", Title: "md", Alive: true, Dir: "/lewkit-room"}}

	model.drawerAt = 2
	model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	assert.Equal(t, "pockets/work", model.heldSession)
	assert.True(t, model.drawer)
	assert.Empty(t, api.moves)

	model.drawerAt = -1
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	applyCmd(model, cmd)
	require.Equal(t, []movedMsg{
		{id: "ed", path: "work/editor"},
		{id: "docs", path: "work/docs"},
	}, api.moves)
	assert.Equal(t, "work/editor", model.byID["ed"].Path)
	assert.Equal(t, "work/docs", model.byID["docs"].Path)
	assert.Equal(t, "/lewkit-room", model.byID["ed"].Dir)
	assert.Equal(t, "work", model.shown)
	assert.Empty(t, model.place)
	assert.Empty(t, model.heldSession)
	assert.True(t, model.drawer)
	assert.Empty(t, api.writes)
}

func TestDrawerRootShowsEveryPane(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = false
	model.drawer = true
	model.shown = "work"
	model.followAll = false
	model.followPrefix = "work"
	model.order = []string{"ed"}
	model.byID["ed"] = Pane{PaneInfo: PaneInfo{ID: "ed", Path: "work/editor", Title: "hx", Alive: true}}

	runCmd(clickRoot(t, model))
	assert.Empty(t, model.shown)
	assert.True(t, model.followAll)
	assert.Empty(t, model.followPrefix)
	assert.True(t, model.drawer)
	assert.Empty(t, api.writes)
	plain, _ := renderPlain(model)
	assert.Contains(t, plain, "* /")
}

func rootShown(plain string) bool {
	for _, line := range strings.Split(plain, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "/" || trim == "* /" {
			return true
		}
	}
	return false
}

func clickRoot(t *testing.T, m *Model) tea.Cmd {
	t.Helper()
	_, lines := renderPlain(m)
	for y, text := range lines {
		trim := strings.TrimSpace(text)
		if trim != "/" && trim != "* /" {
			continue
		}
		at := strings.Index(text, "/")
		_, cmd := m.Update(tea.MouseClickMsg{X: colAt(text, at), Y: y, Button: tea.MouseLeft})
		return cmd
	}
	require.Fail(t, "missing root")
	return nil
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

func clickSessionCarry(t *testing.T, m *Model, row string) tea.Cmd {
	t.Helper()
	_, lines := renderPlain(m)
	for y, text := range lines {
		if !strings.Contains(text, row) || !strings.HasSuffix(strings.TrimRight(text, " "), "x") {
			continue
		}
		at := strings.LastIndex(text, "→")
		if at < 0 {
			continue
		}
		_, cmd := m.Update(tea.MouseClickMsg{X: colAt(text, at), Y: y, Button: tea.MouseLeft})
		return cmd
	}
	require.Fail(t, "missing session carry mark", row)
	return nil
}

func clickCarry(t *testing.T, m *Model, name string) tea.Cmd {
	t.Helper()
	_, lines := renderPlain(m)
	for y, text := range lines {
		if !strings.Contains(text, name) || !strings.HasSuffix(strings.TrimRight(text, " "), "→") {
			continue
		}
		at := strings.LastIndex(text, "→")
		_, cmd := m.Update(tea.MouseClickMsg{X: colAt(text, at), Y: y, Button: tea.MouseLeft})
		return cmd
	}
	require.Fail(t, "missing carry mark", name)
	return nil
}

func applyCmd(m *Model, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	msg := cmd()
	if msg == nil {
		return
	}
	batch, ok := msg.(tea.BatchMsg)
	if ok {
		for _, inner := range batch {
			applyCmd(m, inner)
		}
		return
	}
	_, next := m.Update(msg)
	applyCmd(m, next)
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
