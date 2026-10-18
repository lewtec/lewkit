package term

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-runewidth"
)

const (
	railStartW = 5
	railFlipW  = 8
	recentCap  = 6
)

type hitKind int

const (
	hitStart hitKind = iota
	hitTab
	hitToggle
	hitSession
	hitClose
	hitChoice
	hitChild
	hitNewDir
	hitCarry
	hitFolder
)

type hit struct {
	x, y, w int
	kind    hitKind
	id      string
	index   int
}

type sessionGroup struct {
	key   string
	dir   bool
	panes []string
}

type drawerRow struct {
	session bool
	newDir  bool
	key     string
	pane    string
	text    string
	folder  string
}

type hierRow struct {
	text   string
	pane   string
	folder string
}

type tnode struct {
	name string
	pane string
	kids []*tnode
}

func (m *Model) chrome(origin, height int, record bool) []string {
	width := m.width
	if width < 1 {
		width = 80
	}
	if !m.palette {
		m.shownChoices = nil
		railY := origin
		return []string{m.rail(width, railY, record && railY < height)}
	}
	lines := []string{fit(m.prompt(), width)}
	m.shownChoices = nil
	if m.step != StepLine {
		var rows []string
		switch m.step {
		case StepPanes:
			for _, id := range m.filteredPanes() {
				pane := m.byID[id]
				label := id
				if pane.Path != "" && pane.Path != id {
					label = pane.Path
				}
				rows = append(rows, label+"  "+pane.Title)
			}
		case StepEntries:
			for _, dir := range m.filteredDirs() {
				rows = append(rows, dir+"/")
			}
		default:
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
			m.shownChoices = append(m.shownChoices, i)
			mark := "  "
			if i == cursor {
				mark = "> "
			}
			lines = append(lines, fit(mark+rows[i], width))
		}
	}
	if record {
		for i, idx := range m.shownChoices {
			y := origin + 1 + i
			if y >= height {
				break
			}
			m.hits = append(m.hits, hit{x: 0, y: y, w: width, kind: hitChoice, index: idx})
		}
	}
	railY := origin + len(lines)
	lines = append(lines, m.rail(width, railY, record && railY < height))
	return lines
}

func (m *Model) rail(width, y int, record bool) string {
	if width < railStartW+railFlipW {
		return fit("lew", width)
	}
	start := " lew "
	flip := " drawer "
	if m.drawer {
		flip = "  desk  "
	}
	middle := m.railMiddle(width-railStartW-railFlipW, y, record)
	if record {
		m.hits = append(m.hits, hit{x: 0, y: y, w: railStartW, kind: hitStart})
		m.hits = append(m.hits, hit{x: width - railFlipW, y: y, w: railFlipW, kind: hitToggle})
	}
	return rev(start, m.palette) + middle + rev(flip, m.drawer)
}

func (m *Model) railMiddle(width, y int, record bool) string {
	if width <= 0 {
		return ""
	}
	if m.errText != "" {
		return fit(m.errText, width)
	}
	var ids []string
	for _, id := range m.recent {
		if _, ok := m.byID[id]; ok {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		whisper := "quiet, for now"
		if m.drawer {
			whisper = "sessions live here"
		}
		return fit(whisper, width)
	}
	var b strings.Builder
	used := 0
	for _, id := range ids {
		text := " " + m.tabLabel(id) + " "
		w := runewidth.StringWidth(text)
		if used+w > width {
			if used > 0 {
				break
			}
			text = fit(strings.TrimSpace(text), width)
			w = runewidth.StringWidth(text)
		}
		b.WriteString(rev(text, id == m.focus))
		if record {
			m.hits = append(m.hits, hit{x: railStartW + used, y: y, w: w, kind: hitTab, id: id})
		}
		used += w
		if used >= width {
			break
		}
	}
	if used < width {
		b.WriteString(strings.Repeat(" ", width-used))
	}
	return b.String()
}

func (m *Model) tabLabel(id string) string {
	pane := m.byID[id]
	base := pathBase(panePath(pane))
	if base != "" {
		return base
	}
	if pane.Title != "" {
		return pane.Title
	}
	return id
}

func (m *Model) drawerBody(width, rows int) []string {
	if rows <= 0 {
		return nil
	}
	lines := []string{fit("the drawer", width)}
	if len(lines) < rows {
		lines = append(lines, fit("standing in "+roomLabel(m.room()), width))
	}
	if len(lines) < rows {
		git := m.gitLine
		if git == "" {
			git = "looking around"
		}
		lines = append(lines, fit(git, width))
	}
	if m.held != "" && len(lines) < rows {
		lines = append(lines, fit("carrying "+m.tabLabel(m.held), width))
	}
	if m.place != "" && len(lines) < rows {
		lines = append(lines, fit("pocket "+m.place, width))
	}
	items := m.drawerRows()
	if !m.hasSession() && len(lines) < rows {
		lines = append(lines, fit("nothing in here yet. lew can start a pane.", width))
	}
	if len(items) == 0 {
		m.drawerAt = 0
	} else if m.drawerAt >= len(items) {
		m.drawerAt = len(items) - 1
	} else if m.drawerAt < 0 {
		m.drawerAt = 0
	}
	for i, item := range items {
		if len(lines) >= rows {
			break
		}
		y := len(lines)
		var line string
		switch {
		case item.newDir:
			text := "+ directory"
			if m.naming {
				text = "directory> " + m.name
			}
			line = fit(text, width)
			m.hits = append(m.hits, hit{x: 0, y: y, w: width, kind: hitNewDir})
		case item.session:
			mark := " · "
			if item.key == m.shown {
				mark = " * "
			}
			line = edgeLine(mark+item.key, width)
			if width >= 3 {
				m.hits = append(m.hits, hit{x: 0, y: y, w: width - 3, kind: hitSession, id: item.key})
				m.hits = append(m.hits, hit{x: width - 3, y: y, w: 3, kind: hitClose, id: item.key})
			}
		case item.pane != "":
			line = edgeMark(item.text, " →", width)
			markW := runewidth.StringWidth(" →")
			if markW < width {
				hitW := markW
				if width-markW > 0 {
					hitW++
				}
				m.hits = append(m.hits, hit{x: width - hitW, y: y, w: hitW, kind: hitCarry, id: item.pane})
				m.hits = append(m.hits, hit{x: 0, y: y, w: width - hitW, kind: hitChild, id: item.pane})
			} else {
				m.hits = append(m.hits, hit{x: 0, y: y, w: width, kind: hitChild, id: item.pane})
			}
		default:
			line = fit(item.text, width)
			m.hits = append(m.hits, hit{x: 0, y: y, w: width, kind: hitFolder, id: item.folder})
		}
		if i == m.drawerAt {
			line = rev(line, true)
		}
		lines = append(lines, line)
	}
	for len(lines) < rows {
		lines = append(lines, fit("", width))
	}
	return lines
}

func (m *Model) drawerRows() []drawerRow {
	rows := []drawerRow{{newDir: true, text: "+ directory"}}
	for _, group := range m.sessionGroups() {
		folder := ""
		if group.dir {
			folder = group.key
		}
		rows = append(rows, drawerRow{session: true, key: group.key, folder: folder})
		if group.key != m.shown || !group.dir {
			continue
		}
		for _, row := range m.hierarchy(group.key) {
			rows = append(rows, drawerRow{
				key:    group.key,
				pane:   row.pane,
				text:   row.text,
				folder: row.folder,
			})
		}
	}
	return rows
}

func (m *Model) sessionGroups() []sessionGroup {
	var groups []sessionGroup
	index := map[string]int{}
	for _, id := range m.order {
		pane, ok := m.byID[id]
		if !ok {
			continue
		}
		key := sessionKey(pane)
		dir := strings.Contains(panePath(pane), "/")
		at, seen := index[key]
		if !seen {
			index[key] = len(groups)
			groups = append(groups, sessionGroup{key: key, dir: dir, panes: []string{id}})
			continue
		}
		groups[at].panes = append(groups[at].panes, id)
		if dir {
			groups[at].dir = true
		}
	}
	for _, path := range m.made {
		key := sessionKeyFromPath(path)
		at, seen := index[key]
		if !seen {
			index[key] = len(groups)
			groups = append(groups, sessionGroup{key: key, dir: true})
			continue
		}
		groups[at].dir = true
	}
	return groups
}

func (m *Model) hierarchy(prefix string) []hierRow {
	root := &tnode{}
	for _, id := range m.order {
		pane, ok := m.byID[id]
		if !ok {
			continue
		}
		path := panePath(pane)
		if prefix == "" || !strings.HasPrefix(path, prefix+"/") {
			continue
		}
		rest := strings.TrimPrefix(path, prefix+"/")
		node := root
		parts := strings.Split(rest, "/")
		for i, part := range parts {
			node = node.child(part)
			if i == len(parts)-1 {
				node.pane = id
			}
		}
	}
	for _, path := range m.made {
		if prefix == "" || !strings.HasPrefix(path, prefix+"/") {
			continue
		}
		rest := strings.TrimPrefix(path, prefix+"/")
		node := root
		for _, part := range strings.Split(rest, "/") {
			node = node.child(part)
		}
	}
	var rows []hierRow
	root.walk("", prefix, &rows)
	return rows
}

func (n *tnode) child(name string) *tnode {
	for _, kid := range n.kids {
		if kid.name == name {
			return kid
		}
	}
	kid := &tnode{name: name}
	n.kids = append(n.kids, kid)
	return kid
}

func (n *tnode) walk(prefix, path string, rows *[]hierRow) {
	for i, kid := range n.kids {
		last := i == len(n.kids)-1
		branch := "├ "
		next := prefix + "│ "
		if last {
			branch = "└ "
			next = prefix + "  "
		}
		full := kid.name
		if path != "" {
			full = path + "/" + kid.name
		}
		*rows = append(*rows, hierRow{text: prefix + branch + kid.name, pane: kid.pane, folder: full})
		kid.walk(next, full, rows)
	}
}

func (m *Model) drawerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.naming {
		return m.nameDirectory(msg)
	}
	switch msg.Keystroke() {
	case "up":
		if m.drawerAt > 0 {
			m.drawerAt--
		}
		return m, nil
	case "down":
		n := len(m.drawerRows())
		if n == 0 {
			m.drawerAt = 0
			return m, nil
		}
		if m.drawerAt < n-1 {
			m.drawerAt++
		}
		return m, nil
	case "enter":
		return m.activateDrawer()
	case "esc":
		m.drawer = false
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) activateDrawer() (tea.Model, tea.Cmd) {
	rows := m.drawerRows()
	if m.drawerAt < 0 || m.drawerAt >= len(rows) {
		return m, nil
	}
	row := rows[m.drawerAt]
	if row.newDir {
		if m.naming {
			m.createDirectory(m.name)
			return m, nil
		}
		m.startNaming()
		return m, nil
	}
	if row.pane == "" && row.folder != "" && m.held != "" {
		return m, m.moveInto(row.folder)
	}
	if row.pane == "" && row.folder != "" && !row.session {
		m.place = row.folder
		m.shown = sessionKeyFromPath(row.folder)
		return m, nil
	}
	if row.session {
		if m.held != "" && row.folder != "" {
			return m, m.moveInto(row.folder)
		}
		return m, m.selectSession(row.key)
	}
	m.held = ""
	return m, m.showPane(row.pane)
}

func (m *Model) click(x, y int) (tea.Model, tea.Cmd) {
	for _, item := range m.hits {
		if y == item.y && x >= item.x && x < item.x+item.w {
			return m.useHit(item)
		}
	}
	return m, nil
}

func (m *Model) useHit(item hit) (tea.Model, tea.Cmd) {
	switch item.kind {
	case hitStart:
		m.palette = !m.palette
		return m, nil
	case hitToggle:
		m.drawer = !m.drawer
		if m.drawer {
			return m, m.lookGit()
		}
		return m, nil
	case hitTab:
		return m, m.focusTab(item.id)
	case hitSession:
		if m.held != "" && m.sessionDir(item.id) {
			return m, m.moveInto(item.id)
		}
		return m, m.selectSession(item.id)
	case hitClose:
		return m, m.closeSession(item.id)
	case hitChoice:
		m.cursor = item.index
		return m.activate()
	case hitChild:
		m.held = ""
		return m, m.showPane(item.id)
	case hitNewDir:
		m.startNaming()
		return m, nil
	case hitCarry:
		if m.held == item.id {
			m.held = ""
			return m, nil
		}
		m.held = item.id
		m.drawer = true
		return m, nil
	case hitFolder:
		if m.held != "" {
			return m, m.moveInto(item.id)
		}
		m.place = item.id
		if item.id != "" {
			m.shown = sessionKeyFromPath(item.id)
		}
		m.drawer = true
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) focusTab(id string) tea.Cmd {
	pane, ok := m.byID[id]
	if !ok {
		return nil
	}
	m.drawer = false
	m.palette = false
	m.focus = id
	m.touch(id)
	m.errText = ""
	if m.followAll && m.shown == "" {
		return nil
	}
	key := sessionKey(pane)
	m.shown = key
	if m.sessionDir(key) {
		m.followAll = false
		m.followPrefix = key
		m.follow = nil
		return m.rewatch()
	}
	m.followAll = false
	m.followPrefix = ""
	m.follow = append([]string(nil), m.sessionPanes(key)...)
	return m.rewatch()
}

func (m *Model) selectSession(key string) tea.Cmd {
	group, ok := m.group(key)
	if !ok {
		return nil
	}
	m.shown = key
	m.drawer = true
	m.palette = false
	m.errText = ""
	if group.dir {
		m.followAll = false
		m.followPrefix = key
		m.follow = nil
		if !contains(group.panes, m.focus) && len(group.panes) > 0 {
			m.focus = group.panes[0]
		}
		if contains(group.panes, m.focus) {
			m.touch(m.focus)
		}
		return tea.Batch(m.rewatch(), m.lookGit())
	}
	m.followAll = false
	m.followPrefix = ""
	m.follow = append([]string(nil), group.panes...)
	if len(group.panes) > 0 {
		m.focus = group.panes[0]
		m.touch(m.focus)
	}
	return tea.Batch(m.rewatch(), m.lookGit())
}

func (m *Model) showPane(id string) tea.Cmd {
	if id == "" {
		return nil
	}
	pane, ok := m.byID[id]
	if !ok {
		return nil
	}
	m.focus = id
	m.touch(id)
	m.drawer = false
	m.palette = false
	m.errText = ""
	key := sessionKey(pane)
	m.shown = key
	if strings.Contains(panePath(pane), "/") {
		m.followAll = false
		m.followPrefix = key
		m.follow = nil
		return m.rewatch()
	}
	m.followAll = false
	m.followPrefix = ""
	m.follow = append([]string(nil), m.sessionPanes(key)...)
	return m.rewatch()
}

func (m *Model) closeSession(key string) tea.Cmd {
	ids := append([]string(nil), m.sessionPanes(key)...)
	m.made = withoutPrefix(m.made, key)
	if m.place == key || strings.HasPrefix(m.place, key+"/") {
		m.place = ""
	}
	if len(ids) == 0 {
		if m.shown == key {
			m.shown = ""
			m.followAll = true
			m.followPrefix = ""
			m.follow = nil
		}
		return nil
	}
	for _, id := range ids {
		m.drop(id)
	}
	cmds := make([]tea.Cmd, 0, len(ids)+1)
	for _, id := range ids {
		cmds = append(cmds, m.closePane(id))
	}
	if m.shown == key {
		m.shown = ""
		m.followAll = true
		m.followPrefix = ""
		m.follow = nil
		m.drawerAt = 0
		cmds = append(cmds, m.rewatch())
	}
	return tea.Batch(cmds...)
}

func (m *Model) closePane(id string) tea.Cmd {
	return func() tea.Msg {
		if err := m.api.Remove(m.ctx, id); err != nil {
			return errMsg{err}
		}
		return wroteMsg{}
	}
}

func (m *Model) group(key string) (sessionGroup, bool) {
	for _, group := range m.sessionGroups() {
		if group.key == key {
			return group, true
		}
	}
	return sessionGroup{}, false
}

func (m *Model) sessionDir(key string) bool {
	group, ok := m.group(key)
	return ok && group.dir
}

func (m *Model) sessionPanes(key string) []string {
	group, ok := m.group(key)
	if !ok {
		return nil
	}
	return group.panes
}

func (m *Model) touch(id string) {
	if id == "" {
		return
	}
	next := make([]string, 0, recentCap)
	next = append(next, id)
	for _, item := range m.recent {
		if item == id || len(next) == recentCap {
			continue
		}
		next = append(next, item)
	}
	m.recent = next
}

func (m *Model) drop(id string) {
	delete(m.byID, id)
	m.order = dropID(m.order, id)
	m.recent = dropID(m.recent, id)
	m.follow = dropID(m.follow, id)
	if m.focus == id {
		m.focus = ""
	}
	if m.held == id {
		m.held = ""
	}
}

func (m *Model) room() string {
	if pane, ok := m.byID[m.focus]; ok && pane.Dir != "" {
		return pane.Dir
	}
	return m.cwd
}

func (m *Model) lookGit() tea.Cmd {
	dir := m.room()
	return func() tea.Msg {
		return gitMsg{dir: dir, line: readGit(m.ctx, dir)}
	}
}

func (m *Model) hasSession() bool {
	return len(m.sessionGroups()) > 0
}

func (m *Model) startNaming() {
	m.naming = true
	m.name = ""
	m.errText = ""
	m.drawer = true
}

func (m *Model) nameDirectory(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Keystroke() {
	case "enter":
		m.createDirectory(m.name)
		return m, nil
	case "esc":
		m.naming = false
		m.name = ""
		return m, nil
	case "backspace":
		runes := []rune(m.name)
		if len(runes) == 0 {
			m.naming = false
			return m, nil
		}
		m.name = string(runes[:len(runes)-1])
		return m, nil
	}
	if msg.Text != "" && !msg.Key().Mod.Contains(tea.ModCtrl) && !msg.Key().Mod.Contains(tea.ModAlt) {
		m.name += msg.Text
	}
	return m, nil
}

func (m *Model) createDirectory(name string) {
	parent := m.place
	if parent == "" && m.shown != "" && m.sessionDir(m.shown) {
		parent = m.shown
	}
	path, err := resolvePath(parent, name)
	if err != nil || path == "" {
		m.errText = ErrPath.Error()
		m.naming = false
		m.name = ""
		return
	}
	if m.keyTaken(path, "", true) {
		m.errText = ErrPath.Error()
		m.naming = false
		m.name = ""
		return
	}
	if !contains(m.made, path) {
		m.made = append(m.made, path)
	}
	m.shown = sessionKeyFromPath(path)
	m.place = path
	m.naming = false
	m.name = ""
	m.errText = ""
	m.drawer = true
}

func (m *Model) moveInto(folder string) tea.Cmd {
	id := m.held
	if id == "" || folder == "" {
		m.held = ""
		return nil
	}
	pane, ok := m.byID[id]
	if !ok {
		m.held = ""
		return nil
	}
	base := pathBase(panePath(pane))
	next := folder + "/" + base
	if panePath(pane) == next {
		m.held = ""
		return nil
	}
	if m.keyTaken(next, id, false) {
		m.errText = ErrPath.Error()
		return nil
	}
	m.held = ""
	m.shown = sessionKeyFromPath(folder)
	m.place = folder
	m.drawer = true
	m.palette = false
	m.errText = ""
	return m.move(id, next)
}

// keyTaken reports a path the tree cannot accept.
// folder is an empty directory. A pane may move inside one. A file may not
// occupy the same key as a directory, and a directory may not sit inside a file.
func (m *Model) keyTaken(path, except string, folder bool) bool {
	if path == "" {
		return false
	}
	for _, id := range m.order {
		if id == except {
			continue
		}
		pane, ok := m.byID[id]
		if !ok {
			continue
		}
		have := panePath(pane)
		if have == path || strings.HasPrefix(path, have+"/") {
			return true
		}
		if !folder && strings.HasPrefix(have, path+"/") {
			return true
		}
	}
	for _, made := range m.made {
		if made == path {
			if !folder {
				return true
			}
			continue
		}
		if strings.HasPrefix(path, made+"/") {
			continue
		}
		if !folder && strings.HasPrefix(made, path+"/") {
			return true
		}
	}
	return false
}

func withoutPrefix(paths []string, key string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == key || strings.HasPrefix(path, key+"/") {
			continue
		}
		out = append(out, path)
	}
	return out
}

func panePath(pane Pane) string {
	if pane.Path != "" {
		return pane.Path
	}
	return pane.ID
}

func sessionKey(pane Pane) string {
	return sessionKeyFromPath(panePath(pane))
}

func sessionKeyFromPath(path string) string {
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return path
}

func edgeLine(left string, width int) string {
	return edgeMark(left, " x", width)
}

func edgeMark(left, suffix string, width int) string {
	if width <= 0 {
		return ""
	}
	w := runewidth.StringWidth(suffix)
	if w < 1 || width < w {
		return fit(left, width)
	}
	return fit(left, width-w) + suffix
}

func rev(text string, on bool) string {
	if !on || text == "" {
		return text
	}
	return "\x1b[7m" + text + "\x1b[0m"
}

func dropID(ids []string, id string) []string {
	out := make([]string, 0, len(ids))
	for _, item := range ids {
		if item != id {
			out = append(out, item)
		}
	}
	return out
}
