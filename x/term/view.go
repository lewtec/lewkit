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
	Watch(context.Context, WatchRequest) error
	Next(context.Context) (Event, error)
}

// Run draws the attach view on standard input and output.
// followPane empty follows the ubersession. Otherwise the view follows that
// pane only. The palette is open at the start. ctrl+p opens and closes it.
// Other actions are palette commands; they do not have key bindings yet.
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

// Model is the command-palette attach view.
type Model struct {
	ctx      context.Context
	api      API
	clientID string

	followAll bool
	follow    []string
	focus     string
	order     []string
	byID      map[string]Pane

	width  int
	height int

	palette bool
	step    Step
	action  Action
	query   string
	cursor  int
	errText string
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
		palette:   true,
		step:      StepActions,
		byID:      map[string]Pane{},
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
	return WatchRequest{Panes: append([]string(nil), m.follow...)}
}

// Update applies a key, a resize, or a server record.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyPressMsg:
		return m.key(msg)
	case loadedMsg:
		m.applyList(msg.panes)
		return m, nil
	case Event:
		m.applyEvent(msg)
		return m, m.listen()
	case spawnedMsg:
		m.store(msg.pane)
		if !m.followAll && !contains(m.follow, msg.pane.ID) {
			m.follow = append(m.follow, msg.pane.ID)
		}
		m.focus = msg.pane.ID
		m.errText = ""
		return m, m.rewatch()
	case killedMsg:
		m.errText = ""
		return m, nil
	case wroteMsg:
		m.errText = ""
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
	case "ctrl+p":
		m.palette = !m.palette
		return m, nil
	case "ctrl+c":
		if m.palette {
			return m, tea.Quit
		}
		return m, m.writeFocused([]byte{0x03})
	}
	if m.palette {
		return m.paletteKey(msg)
	}
	return m, m.writeFocused(inputBytes(msg))
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
	case ActionCommand, ActionSend:
		m.action = action
		m.step = StepLine
		m.query = ""
		m.cursor = 0
		return m, nil
	case ActionUbersession:
		m.followAll = true
		m.errText = ""
		return m, m.rewatch()
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
		m.follow = []string{id}
		m.focus = id
	case ActionAlso:
		if m.followAll {
			m.follow = append([]string(nil), m.order...)
			m.followAll = false
		}
		if !contains(m.follow, id) {
			m.follow = append(m.follow, id)
		}
		m.focus = id
	case ActionUnfollow:
		if m.followAll {
			m.follow = append([]string(nil), m.order...)
			m.followAll = false
		}
		m.follow = removeID(m.follow, id)
		if m.focus == id {
			m.focus = ""
		}
	case ActionFocus:
		m.focus = id
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

func (m *Model) spawn(command []string) tea.Cmd {
	rows, cols := m.spawnSize()
	req := CreateRequest{Command: command, Rows: rows, Cols: cols, Client: m.clientID}
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
}

func (m *Model) applyEvent(ev Event) {
	switch ev.Notice {
	case NoticeSession:
		if ev.Session != nil {
			m.order = append([]string(nil), ev.Session.Panes...)
		}
	case NoticePane, NoticePaneClosed:
		if ev.Pane != nil {
			m.store(*ev.Pane)
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
	return m.focus
}

func (m *Model) visible() []string {
	if m.followAll {
		return append([]string(nil), m.order...)
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
		hay := strings.ToLower(id + " " + pane.Title)
		if query == "" || strings.Contains(hay, query) {
			out = append(out, id)
		}
	}
	return out
}

func (m *Model) choiceCount() int {
	switch m.step {
	case StepPanes:
		return len(m.filteredPanes())
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
	return m.tileBody(len(m.visible()))
}

func (m *Model) spawnSize() (int, int) {
	n := len(m.visible()) + 1
	if m.followAll {
		n = len(m.order) + 1
	}
	return m.tileBody(n)
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
	palette := m.paletteText()
	body := height - len(palette)
	if body < 0 {
		body = 0
		palette = palette[:min(len(palette), height)]
	}
	ids := m.visible()
	lines := make([]string, 0, height)
	cursorX, cursorY := 0, 0
	show := false
	if body == 0 || len(ids) == 0 {
		if len(ids) == 0 && body > 0 {
			lines = append(lines, fit("no panes  ctrl+p  New pane", width))
			for len(lines) < body {
				lines = append(lines, fit("", width))
			}
		}
	} else {
		base := body / len(ids)
		extra := body % len(ids)
		y := 0
		focus := shownFocus(ids, m.focus)
		for i, id := range ids {
			tileH := base
			if i < extra {
				tileH++
			}
			if tileH < 1 {
				continue
			}
			pane := m.byID[id]
			focused := id == focus
			lines = append(lines, titleLine(pane, width, focused))
			gridH := tileH - 1
			for row := range gridH {
				lines = append(lines, paintRow(pane, row, width))
			}
			if focused && !m.palette && gridH > 0 {
				show = true
				cursorX = min(pane.CursorCol, width-1)
				cursorY = y + 1 + min(pane.CursorRow, gridH-1)
			}
			y += tileH
		}
	}
	paletteY := len(lines)
	lines = append(lines, palette...)
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	if m.palette {
		show = true
		cursorX = min(2+runewidth.StringWidth(m.query), width-1)
		cursorY = paletteY
		if cursorY >= height {
			cursorY = height - 1
		}
	}
	return strings.Join(lines, "\n"), cursorX, cursorY, show
}

func (m *Model) paletteText() []string {
	width := m.width
	if width < 1 {
		width = 80
	}
	if !m.palette {
		hint := "ctrl+p palette"
		if m.errText != "" {
			hint = m.errText
		}
		return []string{fit(hint, width)}
	}
	var lines []string
	prompt := "> " + m.query
	switch m.step {
	case StepLine:
		if m.action == ActionCommand {
			prompt = "command> " + m.query
		} else {
			prompt = "send> " + m.query
		}
	case StepPanes:
		prompt = "pane> " + m.query
	}
	lines = append(lines, fit(prompt, width))
	if m.step != StepLine {
		var rows []string
		if m.step == StepPanes {
			for _, id := range m.filteredPanes() {
				pane := m.byID[id]
				rows = append(rows, id+"  "+pane.Title)
			}
		} else {
			for _, action := range m.actions() {
				label, detail := actionText(action)
				rows = append(rows, label+"  "+detail)
			}
		}
		cursor := m.cursor
		if len(rows) == 0 {
			cursor = 0
		} else if cursor >= len(rows) {
			cursor = len(rows) - 1
		} else if cursor < 0 {
			cursor = 0
		}
		from, to := windowChoices(cursor, len(rows), 8)
		for i := from; i < to; i++ {
			mark := "  "
			if i == cursor {
				mark = "> "
			}
			lines = append(lines, fit(mark+rows[i], width))
		}
	}
	hint := "ctrl+p closes  esc back  enter runs"
	if m.errText != "" {
		hint = m.errText
	}
	lines = append(lines, fit(hint, width))
	return lines
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

func titleLine(pane Pane, width int, focused bool) string {
	mark := " "
	if focused {
		mark = "*"
	}
	state := "live"
	if !pane.Alive {
		state = fmt.Sprintf("exit %d", pane.Exit)
	}
	text := fmt.Sprintf("%s %s %s %dx%d %s", mark, pane.ID, pane.Title, pane.Cols, pane.Rows, state)
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
	if cell.FG == 0 && cell.BG == 0 && !cell.Bold {
		return ""
	}
	var parts []string
	if cell.Bold {
		parts = append(parts, "1")
	}
	if cell.FG >= 1 && cell.FG <= 8 {
		parts = append(parts, fmt.Sprintf("%d", 29+cell.FG))
	} else if cell.FG >= 9 && cell.FG <= 16 {
		parts = append(parts, fmt.Sprintf("%d", 81+cell.FG))
	}
	if cell.BG >= 1 && cell.BG <= 8 {
		parts = append(parts, fmt.Sprintf("%d", 39+cell.BG))
	} else if cell.BG >= 9 && cell.BG <= 16 {
		parts = append(parts, fmt.Sprintf("%d", 91+cell.BG))
	}
	if len(parts) == 0 {
		return ""
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
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
		return "Also follow", "add a pane beside the ones shown"
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
	default:
		return string(action), ""
	}
}

func inputBytes(msg tea.KeyPressMsg) []byte {
	switch msg.Keystroke() {
	case "enter":
		return []byte{'\r'}
	case "backspace":
		return []byte{0x7f}
	case "tab":
		return []byte{'\t'}
	case "up":
		return []byte("\x1b[A")
	case "down":
		return []byte("\x1b[B")
	case "right":
		return []byte("\x1b[C")
	case "left":
		return []byte("\x1b[D")
	case "home":
		return []byte("\x1b[H")
	case "end":
		return []byte("\x1b[F")
	case "delete":
		return []byte("\x1b[3~")
	}
	if msg.Text != "" && msg.Key().Mod == 0 {
		return []byte(msg.Text)
	}
	if msg.Key().Mod == tea.ModCtrl && msg.Code >= 'a' && msg.Code <= 'z' {
		return []byte{byte(msg.Code - 'a' + 1)}
	}
	return nil
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
