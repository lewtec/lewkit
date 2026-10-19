package term

import (
	"context"
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAPI struct {
	panes      []PaneInfo
	creates    []CreateRequest
	writes     []WriteRequest
	watches    []WatchRequest
	removes    []string
	moves      []movedMsg
	dirCreates []string
	dirRemoves []string
	dirMoves   [][2]string
}

func (f *fakeAPI) Session(context.Context) (SessionInfo, error) {
	return SessionInfo{ID: "session", Panes: paneIDs(f.panes)}, nil
}

func (f *fakeAPI) Panes(context.Context) ([]PaneInfo, error) {
	return append([]PaneInfo(nil), f.panes...), nil
}

func (f *fakeAPI) Pane(context.Context, string) (Pane, error) {
	return Pane{}, ErrPane
}

func (f *fakeAPI) Create(_ context.Context, req CreateRequest) (Pane, error) {
	f.creates = append(f.creates, req)
	return Pane{PaneInfo: PaneInfo{ID: "new", Title: "sh", Rows: req.Rows, Cols: req.Cols, Alive: true}}, nil
}

func (f *fakeAPI) Write(_ context.Context, req WriteRequest) error {
	f.writes = append(f.writes, req)
	return nil
}

func (f *fakeAPI) Kill(context.Context, string) error { return nil }

func (f *fakeAPI) Move(_ context.Context, id, path string) error {
	f.moves = append(f.moves, movedMsg{id: id, path: path})
	return nil
}

func (f *fakeAPI) CreateDir(_ context.Context, path string) error {
	f.dirCreates = append(f.dirCreates, path)
	return nil
}

func (f *fakeAPI) RemoveDir(_ context.Context, path string) error {
	f.dirRemoves = append(f.dirRemoves, path)
	return nil
}

func (f *fakeAPI) MoveDir(_ context.Context, path, dest string) error {
	f.dirMoves = append(f.dirMoves, [2]string{path, dest})
	return nil
}

func (f *fakeAPI) Remove(_ context.Context, id string) error {
	f.removes = append(f.removes, id)
	return nil
}

func (f *fakeAPI) Watch(_ context.Context, req WatchRequest) error {
	f.watches = append(f.watches, req)
	return nil
}

func (f *fakeAPI) Next(ctx context.Context) (Event, error) {
	<-ctx.Done()
	return Event{}, ctx.Err()
}

func TestPaletteNewUsesTileSize(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	assert.True(t, model.followAll)
	assert.Equal(t, WatchRequest{All: true}, model.watchReq())

	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model.palette = true
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	_ = cmd()
	require.Len(t, api.creates, 1)
	assert.Equal(t, 100, api.creates[0].Cols)
	assert.Equal(t, 29, api.creates[0].Rows)
	wd, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, wd, api.creates[0].Dir)
}

func TestClosedPaletteWriteUsesTileSize(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model.order = []string{"pane-1"}
	model.byID["pane-1"] = Pane{PaneInfo: PaneInfo{ID: "pane-1", Title: "sh", Alive: true, Rows: 24, Cols: 80}}
	model.focus = "pane-1"
	model.palette = false

	_, cmd := model.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	require.NotNil(t, cmd)
	_ = cmd()
	require.Len(t, api.writes, 1)
	assert.Equal(t, "a", api.writes[0].Data)
	assert.Equal(t, 100, api.writes[0].Cols)
	assert.Equal(t, 38, api.writes[0].Rows)
	assert.Empty(t, api.creates)
}

func TestPaletteFollowAndUbersession(t *testing.T) {
	api := &fakeAPI{panes: []PaneInfo{
		{ID: "pane-1", Title: "one"},
		{ID: "pane-2", Title: "two"},
	}}
	model := newModel(t.Context(), api, "pane-1")
	assert.Equal(t, WatchRequest{Panes: []string{"pane-1"}}, model.watchReq())
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = true
	model.order = []string{"pane-1", "pane-2"}
	model.byID["pane-1"] = Pane{PaneInfo: api.panes[0]}
	model.byID["pane-2"] = Pane{PaneInfo: api.panes[1]}

	for range 3 {
		model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	}
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	model.Update(cmd())
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	_ = cmd()
	assert.False(t, model.followAll)
	assert.Equal(t, []string{"pane-1"}, model.follow)
	require.NotEmpty(t, api.watches)
	assert.Equal(t, WatchRequest{Panes: []string{"pane-1"}}, api.watches[len(api.watches)-1])

	for _, r := range "uber" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	actions := model.actions()
	require.Len(t, actions, 1)
	assert.Equal(t, ActionUbersession, actions[0])
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	_ = cmd()
	assert.True(t, model.followAll)
	assert.True(t, api.watches[len(api.watches)-1].All)
}

func TestPaletteFolderSetsCreatePath(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.palette = true
	for _, r := range "open folder" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	actions := model.actions()
	require.Len(t, actions, 1)
	assert.Equal(t, ActionFolder, actions[0])
	model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	for _, r := range "work" {
		model.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}
	model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Equal(t, "work", model.folder)

	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	_ = cmd()
	require.Len(t, api.creates, 1)
	assert.Equal(t, "work/", api.creates[0].Path)
}

func TestPaletteCtrlC(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.order = []string{"pane-1"}
	model.byID["pane-1"] = Pane{PaneInfo: PaneInfo{ID: "pane-1", Title: "sh", Alive: true}}
	model.focus = "pane-1"
	model.palette = false
	_, cmd := model.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	_ = cmd()
	require.Len(t, api.writes, 1)
	assert.Equal(t, "\x03", api.writes[0].Data)

	model.palette = true
	_, cmd = model.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	_, ok := cmd().(tea.QuitMsg)
	assert.True(t, ok)
}

func TestFocusedPaneTakesTheEvents(t *testing.T) {
	api := &fakeAPI{}
	model := newModel(t.Context(), api, "")
	model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	model.order = []string{"pane-1"}
	model.byID["pane-1"] = Pane{PaneInfo: PaneInfo{ID: "pane-1", Title: "hx", Alive: true, Rows: 38, Cols: 100}}
	model.focus = "pane-1"
	model.palette = false
	model.render()

	view := model.View()
	assert.Equal(t, tea.MouseModeCellMotion, view.MouseMode)
	assert.True(t, view.ReportFocus)

	_, cmd := model.Update(tea.MouseWheelMsg{X: 2, Y: 3, Button: tea.MouseWheelUp})
	assert.Nil(t, cmd)
	assert.Empty(t, api.writes)

	model.byID["pane-1"] = Pane{PaneInfo: model.byID["pane-1"].PaneInfo, Input: InputAlt}
	_, cmd = model.Update(tea.MouseWheelMsg{X: 2, Y: 3, Button: tea.MouseWheelUp})
	require.NotNil(t, cmd)
	_ = cmd()
	require.Len(t, api.writes, 1)
	assert.Equal(t, "\x1b[A", api.writes[0].Data)
	assert.Equal(t, 38, api.writes[0].Rows)
	assert.Equal(t, 100, api.writes[0].Cols)
	api.writes = nil

	model.byID["pane-1"] = Pane{PaneInfo: model.byID["pane-1"].PaneInfo, Input: InputMouse | InputSGR}
	_, cmd = model.Update(tea.MouseWheelMsg{X: 2, Y: 3, Button: tea.MouseWheelUp})
	require.NotNil(t, cmd)
	_ = cmd()
	_, cmd = model.Update(tea.MouseClickMsg{X: 4, Y: 5, Button: tea.MouseLeft, Mod: tea.ModShift})
	require.NotNil(t, cmd)
	_ = cmd()
	_, cmd = model.Update(tea.MouseReleaseMsg{X: 4, Y: 5, Button: tea.MouseLeft})
	require.NotNil(t, cmd)
	_ = cmd()
	_, cmd = model.Update(tea.MouseMotionMsg{X: 1, Y: 2})
	assert.Nil(t, cmd)
	require.Len(t, api.writes, 3)
	assert.Equal(t, "\x1b[<64;3;3M", api.writes[0].Data)
	assert.Equal(t, "\x1b[<4;5;5M", api.writes[1].Data)
	assert.Equal(t, "\x1b[<0;5;5m", api.writes[2].Data)
	api.writes = nil

	_, cmd = model.Update(tea.MouseClickMsg{X: 4, Y: 5, Button: tea.MouseRight})
	require.NotNil(t, cmd)
	_ = cmd()
	assert.Equal(t, "\x1b[<2;5;5M", api.writes[0].Data)
	api.writes = nil

	pane := model.byID["pane-1"]
	pane.Input = InputMouse | InputSGR | InputAnyMotion
	model.byID["pane-1"] = pane
	view = model.View()
	assert.Equal(t, tea.MouseModeAllMotion, view.MouseMode)
	_, cmd = model.Update(tea.MouseMotionMsg{X: 1, Y: 2})
	require.NotNil(t, cmd)
	_ = cmd()
	assert.Equal(t, "\x1b[<35;2;2M", api.writes[0].Data)
	api.writes = nil

	model.drawer = true
	_, cmd = model.Update(tea.MouseWheelMsg{X: 2, Y: 3, Button: tea.MouseWheelDown})
	assert.Nil(t, cmd)
	_, cmd = model.Update(tea.MouseClickMsg{X: 4, Y: 5, Button: tea.MouseRight})
	assert.Nil(t, cmd)
	assert.Empty(t, api.writes)
	assert.True(t, model.drawer)
	model.drawer = false

	model.render()
	before := len(api.writes)
	clickRail(t, model, "drawer")
	assert.True(t, model.drawer)
	assert.Len(t, api.writes, before)
	model.drawer = false
	model.palette = false

	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	require.NotNil(t, cmd)
	_ = cmd()
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift})
	require.NotNil(t, cmd)
	_ = cmd()
	_, cmd = model.Update(tea.KeyPressMsg{Code: tea.KeyF5})
	require.NotNil(t, cmd)
	_ = cmd()
	_, cmd = model.Update(tea.KeyPressMsg{Text: "x", Code: 'x', Mod: tea.ModAlt})
	require.NotNil(t, cmd)
	_ = cmd()
	require.Len(t, api.writes, 4)
	assert.Equal(t, "\x1b[5~", api.writes[0].Data)
	assert.Equal(t, "\x1b[1;2A", api.writes[1].Data)
	assert.Equal(t, "\x1b[15~", api.writes[2].Data)
	assert.Equal(t, "\x1bx", api.writes[3].Data)
	api.writes = nil

	_, cmd = model.Update(tea.PasteMsg{Content: "pasted"})
	require.NotNil(t, cmd)
	_ = cmd()
	assert.Equal(t, "pasted", api.writes[0].Data)
	api.writes = nil
	pane = model.byID["pane-1"]
	pane.Input |= InputPaste | InputFocus
	model.byID["pane-1"] = pane
	_, cmd = model.Update(tea.PasteMsg{Content: "pasted"})
	require.NotNil(t, cmd)
	_ = cmd()
	assert.Equal(t, "\x1b[200~pasted\x1b[201~", api.writes[0].Data)
	api.writes = nil
	_, cmd = model.Update(tea.FocusMsg{})
	require.NotNil(t, cmd)
	_ = cmd()
	_, cmd = model.Update(tea.BlurMsg{})
	require.NotNil(t, cmd)
	_ = cmd()
	assert.Equal(t, "\x1b[I", api.writes[0].Data)
	assert.Equal(t, "\x1b[O", api.writes[1].Data)
}

func paneIDs(panes []PaneInfo) []string {
	out := make([]string, len(panes))
	for i, pane := range panes {
		out[i] = pane.ID
	}
	return out
}
