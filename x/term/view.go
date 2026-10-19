package term

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
)

// API is the ubersession surface the attach view uses.
// [Client] implements it. The view does not know whether the records came
// from this process or from RPC.
type API interface {
	Session(context.Context) (SessionInfo, error)
	Panes(context.Context) ([]PaneInfo, error)
	Pane(context.Context, string) (Pane, error)
	Create(context.Context, CreateRequest) (Pane, error)
	Write(context.Context, WriteRequest) error
	Kill(context.Context, string) error
	Move(context.Context, string, string) error
	CreateDir(context.Context, string) error
	RemoveDir(context.Context, string) error
	MoveDir(context.Context, string, string) error
	Remove(context.Context, string) error
	Watch(context.Context, WatchRequest) error
	Next(context.Context) (Event, error)
}

// Run draws the attach view on standard input and output.
// followPane empty follows the ubersession. Otherwise the view follows that
// pane only. The command palette stays closed until lew or ctrl+p opens it.
// The rail keeps recent panes. drawer lists sessions. Each session row names
// that session's directory and git status. + directory stores a folder on
// the server, including an empty one. The arrow on a session nests that
// session into a folder.
// The desk shows the focused pane. ctrl+k, and the same chord with super or
// meta, opens the session view. It does not search from the pane. Typing in
// that view filters sessions, panes, and actions. Folders stay open. One
// enter runs an action.
// The drawer takes clicks and the wheel. The wheel moves through the drawer
// the way the arrow keys do. A click or Enter selects a session, opens a
// folder, or arms a pane. The second Enter on that pane or session opens
// it. Right arrow, or the arrow on the row, picks up a pane, a session, or
// a folder. Enter or a click on a folder nests what is carried. / is the
// root. Up lands on it, and Enter or a click there unnests. Enter without a
// carry only selects or opens. A focused pane takes
// mouse, scroll, paste, and the other keys. The rail still takes a click
// that lands on it. Mouse bytes are written when the process asked for them.
// The wheel moves an alternate screen when it did not. A kitty image is
// drawn into the grid. A different protocol level warns on the rail and in
// the drawer, and the attach stays up.
func Run(ctx context.Context, api API, followPane string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	model := newModel(ctx, api, followPane)
	_, err := tea.NewProgram(model, tea.WithContext(ctx), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)).Run()
	if err == nil || errors.Is(err, tea.ErrProgramKilled) || errors.Is(err, tea.ErrInterrupted) {
		return nil
	}
	return err
}

// Model is the attach desk.
type Model struct {
	ctx      context.Context
	api      API
	clientID string

	followAll    bool
	followPrefix string
	follow       []string
	folder       string
	focus        string
	order        []string
	byID         map[string]Pane

	width  int
	height int

	palette      bool
	step         Step
	action       Action
	query        string
	cursor       int
	errText      string
	levelText    string
	drawer       bool
	shown        string
	recent       []string
	hits         []hit
	shownChoices []int
	drawerAt     int
	find         string
	armed        string

	cwd         string
	gitBy       map[string]string
	made        []string
	place       string
	held        string
	heldSession string
	heldFolder  string
	drawerFrom  int
	naming      bool
	name        string
}

func newModel(ctx context.Context, api API, followPane string) *Model {
	id, err := randomID(4)
	if err != nil {
		id = "client"
	}
	model := &Model{
		ctx:       ctx,
		api:       api,
		clientID:  id,
		followAll: followPane == "",
		step:      StepActions,
		byID:      map[string]Pane{},
	}
	if cwd, err := os.Getwd(); err == nil {
		model.cwd = cwd
	}
	if followPane != "" {
		model.follow = []string{followPane}
		model.focus = followPane
	}
	return model
}

type loadedMsg struct {
	info  SessionInfo
	panes []PaneInfo
}

type errMsg struct{ err error }

type spawnedMsg struct{ pane Pane }

type wroteMsg struct{}

type killedMsg struct{ id string }

type movedMsg struct {
	id   string
	path string
}

// Init loads the ubersession and waits for pushes.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.refresh(), m.listen())
}

func (m *Model) refresh() tea.Cmd {
	return func() tea.Msg {
		info, err := m.api.Session(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		panes, err := m.api.Panes(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		if err := m.api.Watch(m.ctx, m.watchReq()); err != nil {
			return errMsg{err}
		}
		return loadedMsg{info: info, panes: panes}
	}
}

func (m *Model) listen() tea.Cmd {
	return func() tea.Msg {
		event, err := m.api.Next(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		return event
	}
}

func (m *Model) watchReq() WatchRequest {
	if m.followAll {
		return WatchRequest{All: true}
	}
	if m.followPrefix != "" && len(m.follow) == 0 {
		return WatchRequest{Prefix: m.followPrefix}
	}
	return WatchRequest{Panes: append([]string(nil), m.follow...)}
}

// Update applies a key, a resize, or a server record.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.MouseClickMsg:
		return m.onMouse(tea.Mouse(msg), mousePress)
	case tea.MouseReleaseMsg:
		return m.onMouse(tea.Mouse(msg), mouseRelease)
	case tea.MouseWheelMsg:
		return m.onMouse(tea.Mouse(msg), mouseWheel)
	case tea.MouseMotionMsg:
		return m.onMouse(tea.Mouse(msg), mouseMotion)
	case tea.PasteMsg:
		if m.drawer || m.palette {
			return m, nil
		}
		return m, m.writeFocused(pasteBytes(msg.Content, m.focusedInput()))
	case tea.FocusMsg:
		if m.drawer || m.palette {
			return m, nil
		}
		return m, m.writeFocused(focusBytes(true, m.focusedInput()))
	case tea.BlurMsg:
		if m.drawer || m.palette {
			return m, nil
		}
		return m, m.writeFocused(focusBytes(false, m.focusedInput()))
	case tea.KeyPressMsg:
		return m.key(msg)
	case loadedMsg:
		m.applyList(msg.panes)
		m.noteDirs(msg.info)
		return m, nil
	case Event:
		m.applyEvent(msg)
		return m, m.listen()
	case spawnedMsg:
		m.store(msg.pane)
		if m.followPrefix == "" && !m.followAll && !contains(m.follow, msg.pane.ID) {
			m.follow = append(m.follow, msg.pane.ID)
		}
		m.focus = msg.pane.ID
		m.touch(msg.pane.ID)
		m.errText = ""
		return m, m.rewatch()
	case killedMsg:
		m.errText = ""
		return m, nil
	case wroteMsg:
		m.errText = ""
		return m, nil
	case movedMsg:
		if pane, ok := m.byID[msg.id]; ok {
			pane.Path = msg.path
			m.byID[msg.id] = pane
		}
		m.errText = ""
		return m, nil
	case gitMsg:
		if m.gitBy == nil {
			m.gitBy = map[string]string{}
		}
		m.gitBy[msg.dir] = msg.line
		return m, nil
	case errMsg:
		if msg.err == nil || errors.Is(msg.err, context.Canceled) {
			return m, tea.Quit
		}
		m.errText = msg.err.Error()
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Keystroke() {
	case "ctrl+k", "super+k", "meta+k":
		return m.openSession()
	case "ctrl+p":
		m.palette = !m.palette
		if m.palette {
			m.drawer = false
		}
		return m, nil
	case "ctrl+c":
		if m.palette {
			return m, tea.Quit
		}
		if m.drawer {
			return m, nil
		}
		return m, m.writeFocused([]byte{0x03})
	}
	if m.drawer {
		return m.drawerKey(msg)
	}
	if m.palette {
		return m.paletteKey(msg)
	}
	return m, m.writeFocused(inputBytes(msg))
}

// openSession shows the session view. The chord does not search from the pane,
// and it does not leave that view.
func (m *Model) openSession() (tea.Model, tea.Cmd) {
	m.palette = false
	if m.drawer {
		return m, nil
	}
	return m, m.setDrawer(true)
}

func (m *Model) paletteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Keystroke() {
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down":
		m.cursor++
		m.clamp(m.choiceCount())
		return m, nil
	case "enter":
		return m.activate()
	case "esc":
		return m.back(), nil
	case "backspace":
		m.deleteRune()
		m.cursor = 0
		return m, nil
	}
	if msg.Text != "" && !msg.Key().Mod.Contains(tea.ModCtrl) && !msg.Key().Mod.Contains(tea.ModAlt) {
		m.query += msg.Text
		m.cursor = 0
	}
	return m, nil
}

func (m *Model) back() *Model {
	if m.step != StepActions {
		m.step = StepActions
		m.query = ""
		m.cursor = 0
		return m
	}
	m.palette = false
	return m
}

func (m *Model) deleteRune() {
	runes := []rune(m.query)
	if len(runes) == 0 {
		if m.step != StepActions {
			m.step = StepActions
		}
		return
	}
	m.query = string(runes[:len(runes)-1])
}

func (m *Model) activate() (tea.Model, tea.Cmd) {
	switch m.step {
	case StepLine:
		return m.applyLine()
	case StepPanes:
		ids := m.filteredPanes()
		m.clamp(len(ids))
		if len(ids) == 0 {
			m.errText = ErrPane.Error()
			return m, nil
		}
		return m.applyPane(ids[m.cursor])
	case StepEntries:
		dirs := m.filteredDirs()
		m.clamp(len(dirs))
		if len(dirs) == 0 {
			m.errText = ErrPath.Error()
			return m, nil
		}
		return m.applyDir(dirs[m.cursor])
	default:
		actions := m.actions()
		m.clamp(len(actions))
		if len(actions) == 0 {
			return m, nil
		}
		return m.applyAction(actions[m.cursor])
	}
}

func (m *Model) applyAction(action Action) (tea.Model, tea.Cmd) {
	switch action {
	case ActionNew:
		return m, m.spawn(nil)
	case ActionCommand, ActionSend, ActionFolder, ActionMove:
		if action == ActionMove && m.focusID() == "" {
			m.errText = ErrPane.Error()
			return m, nil
		}
		m.action = action
		m.step = StepLine
		m.query = ""
		m.cursor = 0
		return m, nil
	case ActionUbersession:
		m.followAll = true
		m.followPrefix = ""
		m.follow = nil
		m.shown = ""
		m.errText = ""
		return m, m.rewatch()
	case ActionUp:
		m.folder = pathParent(m.folder)
		m.errText = ""
		return m, nil
	case ActionFollowFolder:
		if m.folder == "" {
			m.followAll = true
			m.followPrefix = ""
			m.follow = nil
			m.shown = ""
		} else {
			m.followAll = false
			m.followPrefix = m.folder
			m.follow = nil
			m.shown = sessionKeyFromPath(m.folder)
		}
		m.errText = ""
		return m, m.rewatch()
	case ActionEnter:
		m.action = action
		m.step = StepEntries
		m.query = ""
		m.cursor = 0
		return m, nil
	case ActionDetach:
		return m, tea.Quit
	case ActionFollow, ActionAlso, ActionUnfollow, ActionFocus, ActionKill:
		m.action = action
		m.step = StepPanes
		m.query = ""
		m.cursor = 0
		return m, m.loadPanes()
	default:
		return m, nil
	}
}

func (m *Model) applyPane(id string) (tea.Model, tea.Cmd) {
	switch m.action {
	case ActionFollow:
		m.followAll = false
		m.followPrefix = ""
		m.follow = []string{id}
		m.focus = id
		m.touch(id)
	case ActionAlso:
		if m.followAll || m.followPrefix != "" {
			m.follow = m.visible()
			m.followAll = false
			m.followPrefix = ""
		}
		if !contains(m.follow, id) {
			m.follow = append(m.follow, id)
		}
		m.focus = id
		m.touch(id)
	case ActionUnfollow:
		if m.followAll || m.followPrefix != "" {
			m.follow = m.visible()
			m.followAll = false
			m.followPrefix = ""
		}
		m.follow = removeID(m.follow, id)
		if m.focus == id {
			m.focus = ""
		}
	case ActionFocus:
		m.focus = id
		m.touch(id)
		m.step = StepActions
		m.query = ""
		return m, nil
	case ActionKill:
		m.step = StepActions
		m.query = ""
		return m, m.kill(id)
	default:
		m.step = StepActions
		return m, nil
	}
	m.step = StepActions
	m.query = ""
	m.errText = ""
	return m, m.rewatch()
}

func (m *Model) applyLine() (tea.Model, tea.Cmd) {
	text := m.query
	action := m.action
	m.step = StepActions
	m.query = ""
	switch action {
	case ActionCommand:
		fields := strings.Fields(text)
		if len(fields) == 0 {
			m.errText = ErrCommand.Error()
			return m, nil
		}
		return m, m.spawn(fields)
	case ActionFolder:
		folder, err := resolvePath(m.folder, text)
		if err != nil {
			m.errText = err.Error()
			return m, nil
		}
		m.folder = folder
		m.errText = ""
		return m, nil
	case ActionMove:
		id := m.focusID()
		if id == "" {
			m.errText = ErrPane.Error()
			return m, nil
		}
		path, err := resolvePath(m.folder, text)
		if err != nil {
			m.errText = err.Error()
			return m, nil
		}
		if path == "" {
			m.errText = ErrPath.Error()
			return m, nil
		}
		return m, m.move(id, path)
	case ActionSend:
		id := m.focusID()
		if id == "" {
			m.errText = ErrPane.Error()
			return m, nil
		}
		rows, cols := m.focusSize()
		return m, m.write(id, text+"\n", rows, cols)
	default:
		return m, nil
	}
}

func (m *Model) loadPanes() tea.Cmd {
	return func() tea.Msg {
		panes, err := m.api.Panes(m.ctx)
		if err != nil {
			return errMsg{err}
		}
		return loadedMsg{panes: panes}
	}
}

func (m *Model) rewatch() tea.Cmd {
	req := m.watchReq()
	return func() tea.Msg {
		if err := m.api.Watch(m.ctx, req); err != nil {
			return errMsg{err}
		}
		return wroteMsg{}
	}
}

func (m *Model) applyDir(name string) (tea.Model, tea.Cmd) {
	if m.folder == "" {
		m.folder = name
	} else {
		m.folder = m.folder + "/" + name
	}
	m.step = StepActions
	m.query = ""
	m.errText = ""
	return m, nil
}

func (m *Model) spawn(command []string) tea.Cmd {
	rows, cols := m.spawnSize()
	req := CreateRequest{Command: command, Rows: rows, Cols: cols, Client: m.clientID, Dir: m.room()}
	if m.folder != "" {
		req.Path = m.folder + "/"
	}
	return func() tea.Msg {
		pane, err := m.api.Create(m.ctx, req)
		if err != nil {
			return errMsg{err}
		}
		return spawnedMsg{pane: pane}
	}
}

func (m *Model) write(id, data string, rows, cols int) tea.Cmd {
	req := WriteRequest{ID: id, Data: data, Rows: rows, Cols: cols, Client: m.clientID}
	return func() tea.Msg {
		if err := m.api.Write(m.ctx, req); err != nil {
			return errMsg{err}
		}
		return wroteMsg{}
	}
}

func (m *Model) onMouse(ms tea.Mouse, kind int) (tea.Model, tea.Cmd) {
	if m.drawer {
		if kind == mouseWheel {
			return m.scrollDrawer(ms.Button)
		}
		if kind == mousePress && ms.Button == tea.MouseLeft {
			return m.click(ms.X, ms.Y)
		}
		return m, nil
	}
	if m.palette {
		if kind == mousePress && ms.Button == tea.MouseLeft {
			return m.click(ms.X, ms.Y)
		}
		return m, nil
	}
	if kind == mousePress && ms.Button == tea.MouseLeft && m.hitAt(ms.X, ms.Y) {
		return m.click(ms.X, ms.Y)
	}
	col, row, ok := m.gridPoint(ms.X, ms.Y, kind)
	if !ok {
		return m, nil
	}
	return m, m.writeFocused(encodeMouse(ms, kind, col, row, m.focusedInput()))
}

func (m *Model) hitAt(x, y int) bool {
	for _, item := range m.hits {
		if y == item.y && x >= item.x && x < item.x+item.w {
			return true
		}
	}
	return false
}

func (m *Model) gridPoint(x, y, kind int) (col, row int, ok bool) {
	rows, cols := m.focusSize()
	if rows < 1 {
		rows = 1
	}
	if cols < 1 {
		cols = 1
	}
	inside := x >= 0 && x < cols && y >= 1 && y <= rows
	if kind == mouseWheel && y == 0 && x >= 0 && x < cols {
		inside = true
		y = 1
	}
	if (kind == mousePress || kind == mouseWheel) && !inside {
		return 0, 0, false
	}
	if x < 0 {
		x = 0
	}
	if x >= cols {
		x = cols - 1
	}
	if y < 1 {
		y = 1
	}
	if y > rows {
		y = rows
	}
	return x + 1, y, true
}

func (m *Model) focusedInput() uint32 {
	id := shownFocus(m.visible(), m.focus)
	if id == "" {
		return 0
	}
	return m.byID[id].Input
}

func (m *Model) writeFocused(data []byte) tea.Cmd {
	if len(data) == 0 {
		return nil
	}
	id := m.focusID()
	if id == "" {
		m.errText = ErrPane.Error()
		return nil
	}
	rows, cols := m.focusSize()
	return m.write(id, string(data), rows, cols)
}

func (m *Model) move(id, path string) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.Move(m.ctx, id, path); err != nil {
			return errMsg{err}
		}
		return movedMsg{id: id, path: path}
	}
}

func (m *Model) createDir(path string) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.CreateDir(m.ctx, path); err != nil {
			return errMsg{err}
		}
		return wroteMsg{}
	}
}

func (m *Model) removeDir(path string) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.RemoveDir(m.ctx, path); err != nil {
			return errMsg{err}
		}
		return wroteMsg{}
	}
}

func (m *Model) moveDir(path, dest string) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.MoveDir(m.ctx, path, dest); err != nil {
			return errMsg{err}
		}
		return wroteMsg{}
	}
}

func (m *Model) kill(id string) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.Kill(m.ctx, id); err != nil {
			return errMsg{err}
		}
		return killedMsg{id: id}
	}
}

func (m *Model) applyList(panes []PaneInfo) {
	order := make([]string, 0, len(panes))
	for _, info := range panes {
		order = append(order, info.ID)
		prev := m.byID[info.ID]
		prev.PaneInfo = info
		m.byID[info.ID] = prev
	}
	m.order = order
	if m.focus == "" && len(order) > 0 {
		m.focus = order[0]
	}
	if m.focus != "" {
		m.touch(m.focus)
	}
}

func spokenLevel(level uint32) uint32 {
	if level == 0 {
		return 1
	}
	return level
}

func levelWarning(server uint32) string {
	spoken := spokenLevel(server)
	if spoken == ProtocolLevel {
		return ""
	}
	text := fmt.Sprintf("server protocol %d, client protocol %d", spoken, ProtocolLevel)
	if spoken < ProtocolLevel {
		text += ". restart the server"
	}
	return text
}

func (m *Model) noteDirs(info SessionInfo) {
	m.levelText = levelWarning(info.Level)
	if info.Level == ProtocolLevel {
		m.made = append([]string(nil), info.Dirs...)
	}
}

func (m *Model) applySession(info SessionInfo) {
	if info.Panes != nil {
		m.order = append([]string(nil), info.Panes...)
	}
	m.noteDirs(info)
}

func (m *Model) applyEvent(ev Event) {
	switch ev.Notice {
	case NoticeSession:
		if ev.Session != nil {
			m.applySession(*ev.Session)
		}
	case NoticePane:
		if ev.Pane != nil {
			m.store(*ev.Pane)
		}
	case NoticePaneClosed:
		if ev.Pane != nil {
			m.drop(ev.Pane.ID)
		}
	}
}

func (m *Model) store(pane Pane) {
	m.byID[pane.ID] = pane
	if !contains(m.order, pane.ID) {
		m.order = append(m.order, pane.ID)
	}
}

func (m *Model) focusID() string {
	if pane, ok := m.byID[m.focus]; ok && contains(m.visible(), pane.ID) {
		return pane.ID
	}
	visible := m.visible()
	if len(visible) == 0 {
		return ""
	}
	m.focus = visible[0]
	m.touch(m.focus)
	return m.focus
}

func (m *Model) visible() []string {
	if m.followAll {
		return append([]string(nil), m.order...)
	}
	if m.followPrefix != "" && len(m.follow) == 0 {
		var out []string
		for _, id := range m.order {
			pane := m.byID[id]
			path := pane.Path
			if path == "" {
				path = id
			}
			if pathUnder(path, m.followPrefix) {
				out = append(out, id)
			}
		}
		return out
	}
	var out []string
	for _, id := range m.follow {
		if _, ok := m.byID[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

func (m *Model) actions() []Action {
	query := strings.ToLower(strings.TrimSpace(m.query))
	var out []Action
	for _, action := range ActionNew.Values() {
		label, detail := actionText(action)
		hay := strings.ToLower(label + " " + detail + " " + string(action))
		if query == "" || strings.Contains(hay, query) {
			out = append(out, action)
		}
	}
	return out
}

func (m *Model) filteredPanes() []string {
	query := strings.ToLower(strings.TrimSpace(m.query))
	var out []string
	for _, id := range m.order {
		pane := m.byID[id]
		hay := strings.ToLower(id + " " + pane.Title + " " + pane.Path)
		if query == "" || strings.Contains(hay, query) {
			out = append(out, id)
		}
	}
	return out
}

func (m *Model) filteredDirs() []string {
	query := strings.ToLower(strings.TrimSpace(m.query))
	dirs := childDirs(m.order, func(id string) string {
		return m.byID[id].Path
	}, m.folder)
	if query == "" {
		return dirs
	}
	var out []string
	for _, dir := range dirs {
		if strings.Contains(strings.ToLower(dir), query) {
			out = append(out, dir)
		}
	}
	return out
}

func (m *Model) choiceCount() int {
	switch m.step {
	case StepPanes:
		return len(m.filteredPanes())
	case StepEntries:
		return len(m.filteredDirs())
	case StepLine:
		return 0
	default:
		return len(m.actions())
	}
}

func (m *Model) clamp(n int) {
	if n <= 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= n {
		m.cursor = n - 1
	}
}

func (m *Model) focusSize() (int, int) {
	return m.tileBody(1)
}

func (m *Model) spawnSize() (int, int) {
	return m.tileBody(1)
}

func (m *Model) tileBody(n int) (int, int) {
	if m.width < 2 || m.height < 2 {
		return 24, 80
	}
	if n < 1 {
		n = 1
	}
	body := m.height - len(m.paletteText())
	if body < 1 {
		body = 1
	}
	height := body / n
	if height < 1 {
		height = 1
	}
	rows := height - 1
	if rows < 1 {
		rows = 1
	}
	return rows, m.width
}

// View draws followed panes and the command palette.
func (m *Model) View() tea.View {
	content, x, y, show := m.render()
	view := tea.NewView(content)
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	view.ReportFocus = true
	if !m.drawer && !m.palette && m.focusedInput()&InputAnyMotion != 0 {
		view.MouseMode = tea.MouseModeAllMotion
	}
	view.WindowTitle = "lewkit term"
	if show {
		view.Cursor = tea.NewCursor(x, y)
	}
	return view
}

func (m *Model) render() (string, int, int, bool) {
	width := m.width
	height := m.height
	if width < 1 {
		width = 80
	}
	if height < 1 {
		height = 24
	}
	m.hits = nil
	palette := m.paletteText()
	body := height - len(palette)
	if body < 0 {
		body = 0
	}
	ids := m.visible()
	lines := make([]string, 0, height)
	cursorX, cursorY := 0, 0
	show := false
	switch {
	case m.drawer && body > 0:
		lines = append(lines, m.drawerBody(width, body)...)
	case body == 0 || len(ids) == 0:
		if !m.drawer && len(ids) == 0 && body > 0 {
			lines = append(lines, fit("the desk is clear. lew starts a pane, the drawer keeps sessions.", width))
			for len(lines) < body {
				lines = append(lines, fit("", width))
			}
		}
	default:
		id := shownFocus(ids, m.focus)
		pane := m.byID[id]
		lines = append(lines, titleLine(pane, width, true))
		gridH := body - 1
		for row := range gridH {
			lines = append(lines, paintRow(pane, row, width))
		}
		if !m.palette && gridH > 0 {
			show = true
			cursorX = min(pane.CursorCol, width-1)
			cursorY = 1 + min(pane.CursorRow, gridH-1)
		}
	}
	paletteY := len(lines)
	palette = m.chrome(paletteY, height, true)
	if body == 0 && len(palette) > height {
		palette = palette[:height]
	}
	lines = append(lines, palette...)
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	if m.palette {
		show = true
		prefix := strings.TrimSuffix(m.prompt(), m.query)
		cursorX = min(runewidth.StringWidth(prefix)+runewidth.StringWidth(m.query), width-1)
		cursorY = paletteY
		if cursorY >= height {
			cursorY = height - 1
		}
	}
	return strings.Join(lines, "\n"), cursorX, cursorY, show
}

func (m *Model) paletteText() []string {
	return m.chrome(0, 0, false)
}

func windowChoices(cursor, n, limit int) (int, int) {
	if n == 0 {
		return 0, 0
	}
	if n <= limit {
		return 0, n
	}
	from := cursor - limit/2
	if from < 0 {
		from = 0
	}
	to := from + limit
	if to > n {
		to = n
		from = to - limit
	}
	return from, to
}

func (m *Model) prompt() string {
	switch {
	case m.step == StepLine && m.action == ActionCommand:
		return "command> " + m.query
	case m.step == StepLine && m.action == ActionSend:
		return "send> " + m.query
	case m.step == StepLine && m.action == ActionFolder:
		return "folder> " + m.query
	case m.step == StepLine && m.action == ActionMove:
		return "path> " + m.query
	case m.step == StepPanes:
		return "pane> " + m.query
	case m.step == StepEntries:
		return "folder> " + m.query
	default:
		return "> " + m.query
	}
}

func (m *Model) folderLabel() string {
	if m.folder == "" {
		return "/"
	}
	return "/" + m.folder
}

func titleLine(pane Pane, width int, focused bool) string {
	mark := "·"
	if focused {
		mark = "*"
	}
	state := "live"
	if !pane.Alive {
		state = fmt.Sprintf("exit %d", pane.Exit)
	}
	label := pane.ID
	if pane.Path != "" && pane.Path != pane.ID {
		label = pane.Path
	}
	text := fmt.Sprintf("%s %s %s %dx%d %s", mark, label, pane.Title, pane.Cols, pane.Rows, state)
	return "\x1b[7m" + fit(text, width) + "\x1b[0m"
}

func paintRow(pane Pane, row, width int) string {
	if pane.Cols <= 0 || row >= pane.Rows || len(pane.Cells) < (row+1)*pane.Cols {
		return fit("", width)
	}
	start := row * pane.Cols
	cells := pane.Cells[start : start+pane.Cols]
	var b strings.Builder
	col := 0
	for _, cell := range cells {
		if col >= width {
			break
		}
		if cell.Wide {
			continue
		}
		b.WriteString(sgrOpen(cell))
		if cell.Rune == "" {
			b.WriteByte(' ')
			col++
		} else {
			b.WriteString(cell.Rune)
			w := runewidth.StringWidth(cell.Rune)
			if w < 1 {
				w = 1
			}
			col += w
		}
		b.WriteString("\x1b[0m")
	}
	if col < width {
		b.WriteString(strings.Repeat(" ", width-col))
	}
	return b.String()
}

func sgrOpen(cell Cell) string {
	var parts []string
	if cell.Bold {
		parts = append(parts, "1")
	}
	if cell.Reverse {
		parts = append(parts, "7")
	}
	parts = append(parts, colorCodes(true, cell.FGMode, cell.FG, cell.FGIndex, cell.FGRGB)...)
	parts = append(parts, colorCodes(false, cell.BGMode, cell.BG, cell.BGIndex, cell.BGRGB)...)
	if len(parts) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
}

func colorCodes(fg bool, mode, palette, index int, rgb uint32) []string {
	which := "38"
	if !fg {
		which = "48"
	}
	switch mode {
	case ColorRGB:
		return []string{
			which, "2",
			fmt.Sprintf("%d", (rgb>>16)&0xff),
			fmt.Sprintf("%d", (rgb>>8)&0xff),
			fmt.Sprintf("%d", rgb&0xff),
		}
	case ColorIndex:
		return []string{which, "5", fmt.Sprintf("%d", index)}
	default:
		return paletteCodes(fg, palette)
	}
}

func paletteCodes(fg bool, palette int) []string {
	if palette >= 1 && palette <= 8 {
		base := 29
		if !fg {
			base = 39
		}
		return []string{fmt.Sprintf("%d", base+palette)}
	}
	if palette >= 9 && palette <= 16 {
		base := 81
		if !fg {
			base = 91
		}
		return []string{fmt.Sprintf("%d", base+palette)}
	}
	return nil
}

func fit(text string, width int) string {
	if width <= 0 {
		return ""
	}
	var b strings.Builder
	used := 0
	for _, r := range text {
		w := runewidth.RuneWidth(r)
		if w < 0 {
			w = 0
		}
		if used+w > width {
			break
		}
		b.WriteRune(r)
		used += w
	}
	if used < width {
		b.WriteString(strings.Repeat(" ", width-used))
	}
	return b.String()
}

func actionText(action Action) (string, string) {
	switch action {
	case ActionNew:
		return "New pane", "open a shell"
	case ActionCommand:
		return "New command", "open a pane running a command"
	case ActionUbersession:
		return "Attach ubersession", "follow every pane"
	case ActionFollow:
		return "Follow pane", "show only that pane"
	case ActionAlso:
		return "Also follow", "watch that pane too"
	case ActionUnfollow:
		return "Unfollow pane", "stop showing a pane"
	case ActionFocus:
		return "Focus pane", "write to this pane"
	case ActionSend:
		return "Send line", "write a line to the focused pane"
	case ActionKill:
		return "Kill pane", "stop the pane process"
	case ActionDetach:
		return "Detach", "close this client"
	case ActionFolder:
		return "Open folder", "put new panes in this folder"
	case ActionUp:
		return "Folder up", "leave this folder"
	case ActionEnter:
		return "Enter folder", "step into a folder that holds panes"
	case ActionMove:
		return "Move pane", "give the focused pane a new key"
	case ActionFollowFolder:
		return "Follow folder", "watch the panes in this folder"
	default:
		return string(action), ""
	}
}

func shownFocus(ids []string, focus string) string {
	if contains(ids, focus) {
		return focus
	}
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func contains(ids []string, id string) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

func removeID(ids []string, id string) []string {
	out := ids[:0]
	for _, item := range ids {
		if item != id {
			out = append(out, item)
		}
	}
	return out
}
