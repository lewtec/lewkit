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
	hitCarrySession
	hitAction
	hitRoot
)

type hit struct {
	x, y, w int
	kind    hitKind
	id      string
	index   int
}

type sessionGroup struct {
	key    string
	name   string
	parent string
	dir    bool
	panes  []string
}

type drawerRow struct {
	session bool
	pocket  bool
	newDir  bool
	key     string
	name    string
	indent  string
	pane    string
	text    string
	folder  string
	action  Action
	act     bool
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
	return paint(start, m.palette) + middle + paint(flip, m.drawer)
}

func (m *Model) railMiddle(width, y int, record bool) string {
	if width <= 0 {
		return ""
	}
	if m.errText != "" {
		return styleErr.Render(fit(m.errText, width))
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
		return styleMuted.Render(fit(whisper, width))
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
		b.WriteString(paint(text, id == m.focus))
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
	title := "the drawer"
	if q := strings.TrimSpace(m.find); q != "" {
		title += "  " + q
	}
	lines := []string{styleTitle.Render(fit(title, width))}
	switch {
	case m.heldSession != "" && len(lines) < rows:
		lines = append(lines, styleMuted.Render(fit("carrying "+pathBase(m.heldSession), width)))
	case m.heldFolder != "" && len(lines) < rows:
		lines = append(lines, styleMuted.Render(fit("carrying "+pathBase(m.heldFolder), width)))
	case m.held != "" && len(lines) < rows:
		lines = append(lines, styleMuted.Render(fit("carrying "+m.tabLabel(m.held), width)))
	}
	if m.place != "" && len(lines) < rows {
		lines = append(lines, styleMuted.Render(fit("pocket "+m.place, width)))
	}
	if len(lines) < rows {
		label := " /"
		if m.shown == "" {
			label = "* /"
		}
		y := len(lines)
		m.hits = append(m.hits, hit{x: 0, y: y, w: width, kind: hitRoot})
		lines = append(lines, dress(fit(label, width), m.drawerAt < 0 || m.shown == "", false))
	}
	items := m.drawerRows()
	if !m.hasSession() && !m.finding() && len(lines) < rows {
		lines = append(lines, styleMuted.Render(fit("nothing in here yet. lew can start a pane.", width)))
	}
	if m.finding() && len(items) == 0 && len(lines) < rows {
		lines = append(lines, styleMuted.Render(fit("nothing matches", width)))
	}
	if len(items) == 0 {
		m.drawerAt = -1
		m.drawerFrom = 0
	} else if m.drawerAt >= len(items) {
		m.drawerAt = len(items) - 1
	}
	room := rows - len(lines)
	if room < 1 || len(items) == 0 {
		for len(lines) < rows {
			lines = append(lines, fit("", width))
		}
		return lines
	}
	if m.drawerAt < 0 {
		m.drawerFrom = 0
	} else if m.drawerFrom < 0 || m.drawerAt < m.drawerFrom {
		m.drawerFrom = m.drawerAt
	}
	if m.drawerAt >= m.drawerFrom+room {
		m.drawerFrom = m.drawerAt - room + 1
	}
	if maxFrom := len(items) - room; m.drawerFrom > maxFrom {
		m.drawerFrom = maxFrom
	}
	if m.drawerFrom < 0 {
		m.drawerFrom = 0
	}
	for i := m.drawerFrom; i < len(items); i++ {
		if len(lines) >= rows {
			break
		}
		item := items[i]
		y := len(lines)
		var line string
		if m.finding() && !item.session && !item.pocket {
			line = fit(item.text, width)
			switch {
			case item.act:
				m.hits = append(m.hits, hit{x: 0, y: y, w: width, kind: hitAction, index: i})
			case item.pane != "":
				m.hits = append(m.hits, hit{x: 0, y: y, w: width, kind: hitChild, id: item.pane})
			}
			line = dress(line, i == m.drawerAt, item.pane != "" && item.pane == m.armed)
			lines = append(lines, line)
			continue
		}
		switch {
		case item.newDir:
			text := "+ directory"
			if m.naming {
				text = "directory> " + m.name
			}
			line = fit(text, width)
			m.hits = append(m.hits, hit{x: 0, y: y, w: width, kind: hitNewDir})
		case item.session:
			mark := "· "
			if item.key == m.shown {
				mark = "* "
			}
			if item.indent == "" {
				mark = " " + mark
			}
			head := item.indent + mark + item.name
			cwd := ""
			git := ""
			if dir := m.sessionRoom(item.key); dir != "" {
				cwd = roomLabel(dir)
				git = m.gitStatus(dir)
				if git == "" {
					git = "looking around"
				}
			}
			const suffix = " → x"
			label := sessionLabel(head, cwd, git, width-runewidth.StringWidth(suffix))
			line = edgeMark(label, suffix, width)
			sufW := runewidth.StringWidth(suffix)
			if sufW < width {
				arrowAt := strings.Index(suffix, "→")
				arrowCol := width - sufW + runewidth.StringWidth(suffix[:arrowAt])
				xAt := strings.LastIndex(suffix, "x")
				xCol := width - sufW + runewidth.StringWidth(suffix[:xAt])
				m.hits = append(m.hits, hit{x: arrowCol, y: y, w: runewidth.RuneWidth('→'), kind: hitCarrySession, id: item.key})
				m.hits = append(m.hits, hit{x: xCol, y: y, w: width - xCol, kind: hitClose, id: item.key})
				m.hits = append(m.hits, hit{x: 0, y: y, w: arrowCol, kind: hitSession, id: item.key})
			}
		case item.pocket:
			line = edgeLine(item.text, width)
			if width >= 3 {
				m.hits = append(m.hits, hit{x: 0, y: y, w: width - 3, kind: hitFolder, id: item.folder})
				m.hits = append(m.hits, hit{x: width - 3, y: y, w: 3, kind: hitClose, id: item.folder})
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
		line = dress(line, i == m.drawerAt, item.pane != "" && item.pane == m.armed)
		lines = append(lines, line)
	}
	for len(lines) < rows {
		lines = append(lines, fit("", width))
	}
	return lines
}

func (m *Model) drawerRows() []drawerRow {
	if m.finding() {
		return m.searchRows()
	}
	rows := []drawerRow{{newDir: true, text: "+ directory"}}
	m.writeLevel("", "", &rows)
	return rows
}

func (m *Model) finding() bool {
	return strings.TrimSpace(m.find) != ""
}

func (m *Model) searchRows() []drawerRow {
	query := strings.ToLower(strings.TrimSpace(m.find))
	var rows []drawerRow
	for _, group := range m.sessionGroups() {
		if !m.sessionMatches(group, query) {
			continue
		}
		folder := ""
		if group.dir {
			folder = group.key
		}
		rows = append(rows, drawerRow{
			session: true,
			key:     group.key,
			name:    group.name,
			folder:  folder,
		})
	}
	for _, path := range m.made {
		if !strings.Contains(strings.ToLower(path), query) {
			continue
		}
		rows = append(rows, drawerRow{pocket: true, key: path, folder: path, text: " · " + pathBase(path)})
	}
	for _, id := range m.order {
		pane, ok := m.byID[id]
		if !ok {
			continue
		}
		label := m.tabLabel(id)
		hay := strings.ToLower(id + " " + pane.Title + " " + pane.Path + " " + label)
		if !strings.Contains(hay, query) {
			continue
		}
		text := pane.Path
		if text == "" {
			text = id
		}
		if pane.Title != "" && !strings.Contains(strings.ToLower(text), strings.ToLower(pane.Title)) {
			text += "  " + pane.Title
		}
		rows = append(rows, drawerRow{pane: id, text: text, name: label})
	}
	for _, action := range ActionNew.Values() {
		label, detail := actionText(action)
		hay := strings.ToLower(label + " " + detail + " " + string(action))
		if !strings.Contains(hay, query) {
			continue
		}
		rows = append(rows, drawerRow{act: true, action: action, text: label + "  " + detail})
	}
	return rows
}

func (m *Model) sessionMatches(group sessionGroup, query string) bool {
	hay := strings.ToLower(group.name + " " + group.key)
	if dir := m.sessionRoom(group.key); dir != "" {
		hay += " " + strings.ToLower(roomLabel(dir))
		if git := m.gitStatus(dir); git != "" {
			hay += " " + strings.ToLower(git)
		}
	}
	return strings.Contains(hay, query)
}

func (m *Model) writeLevel(parent, indent string, rows *[]drawerRow) {
	nodes := m.levelNodes(parent)
	for i, node := range nodes {
		last := i == len(nodes)-1
		if node.pocket {
			text := node.name
			if parent == "" {
				mark := " · "
				if m.shown == node.path {
					mark = " * "
				}
				text = mark + node.name
			} else {
				branch := "├ "
				if last {
					branch = "└ "
				}
				text = indent + branch + node.name
			}
			*rows = append(*rows, drawerRow{pocket: true, key: node.path, folder: node.path, text: text})
			child := "  "
			if parent != "" {
				child = indent + "│ "
				if last {
					child = indent + "  "
				}
			}
			m.writeLevel(node.path, child, rows)
			continue
		}
		group := node.group
		folder := ""
		if group.dir {
			folder = group.key
		}
		sessIndent := ""
		if parent != "" {
			sessIndent = indent
		}
		*rows = append(*rows, drawerRow{
			session: true,
			key:     group.key,
			name:    group.name,
			indent:  sessIndent,
			folder:  folder,
		})
		if !group.dir {
			continue
		}
		pad := ""
		if parent != "" {
			pad = indent + "  "
		}
		for _, row := range m.hierarchy(group.key) {
			*rows = append(*rows, drawerRow{
				key:    group.key,
				pane:   row.pane,
				text:   pad + row.text,
				folder: row.folder,
			})
		}
	}
}

func (m *Model) levelNodes(parent string) []treeNode {
	var nodes []treeNode
	seen := map[string]bool{}
	for _, path := range m.made {
		if m.madeParent(path) != parent || seen[path] {
			continue
		}
		seen[path] = true
		nodes = append(nodes, treeNode{pocket: true, path: path, name: pathBase(path)})
	}
	for _, group := range m.sessionGroups() {
		if group.parent != parent {
			continue
		}
		nodes = append(nodes, treeNode{name: group.name, group: group})
	}
	return nodes
}

type treeNode struct {
	pocket bool
	path   string
	name   string
	group  sessionGroup
}

func (m *Model) sessionGroups() []sessionGroup {
	var groups []sessionGroup
	index := map[string]int{}
	for _, id := range m.order {
		pane, ok := m.byID[id]
		if !ok {
			continue
		}
		key, name, parent := m.sessionOf(panePath(pane))
		if key == "" {
			continue
		}
		dir := panePath(pane) != key
		at, seen := index[key]
		if !seen {
			index[key] = len(groups)
			groups = append(groups, sessionGroup{
				key:    key,
				name:   name,
				parent: parent,
				dir:    dir,
				panes:  []string{id},
			})
			continue
		}
		groups[at].panes = append(groups[at].panes, id)
		if dir {
			groups[at].dir = true
		}
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
		m.moveDrawer(-1)
		return m, nil
	case "down":
		m.moveDrawer(1)
		return m, nil
	case "right":
		m.grabHighlight()
		return m, nil
	case "enter":
		return m.activateDrawer()
	case "esc":
		if m.carrying() {
			m.clearHold()
			return m, nil
		}
		if m.find != "" {
			m.find = ""
			m.armed = ""
			m.drawerAt = 0
			return m, nil
		}
		m.drawer = false
		m.armed = ""
		return m, nil
	case "backspace":
		m.deleteFind()
		return m, nil
	}
	mod := msg.Key().Mod
	if msg.Text != "" && !mod.Contains(tea.ModCtrl) && !mod.Contains(tea.ModAlt) && !mod.Contains(tea.ModMeta) && !mod.Contains(tea.ModSuper) && !mod.Contains(tea.ModHyper) {
		m.find += msg.Text
		m.armed = ""
		m.drawerAt = 0
	}
	return m, nil
}

func (m *Model) deleteFind() {
	runes := []rune(m.find)
	if len(runes) == 0 {
		return
	}
	m.find = string(runes[:len(runes)-1])
	m.armed = ""
	m.drawerAt = 0
}

func (m *Model) setDrawer(open bool) tea.Cmd {
	m.drawer = open
	m.find = ""
	m.armed = ""
	m.naming = false
	m.name = ""
	if !open {
		return nil
	}
	m.palette = false
	m.drawerAt = 0
	m.drawerFrom = 0
	return m.lookGit()
}

func (m *Model) moveDrawer(delta int) {
	n := len(m.drawerRows())
	if n == 0 {
		m.drawerAt = -1
		m.armed = ""
		return
	}
	m.drawerAt += delta
	if m.drawerAt < -1 {
		m.drawerAt = -1
	}
	if m.drawerAt > n-1 {
		m.drawerAt = n - 1
	}
	m.armed = ""
}

func (m *Model) scrollDrawer(button tea.MouseButton) (tea.Model, tea.Cmd) {
	switch button {
	case tea.MouseWheelUp:
		m.moveDrawer(-1)
	case tea.MouseWheelDown:
		m.moveDrawer(1)
	}
	return m, nil
}

func (m *Model) activateDrawer() (tea.Model, tea.Cmd) {
	if m.drawerAt < 0 {
		return m, m.activateRoot()
	}
	rows := m.drawerRows()
	if m.drawerAt >= len(rows) {
		return m, nil
	}
	row := rows[m.drawerAt]
	if row.act {
		return m.runSearchAction(row.action)
	}
	if row.newDir {
		if m.naming {
			m.createDirectory(m.name)
			return m, nil
		}
		m.startNaming()
		return m, nil
	}
	if row.pocket || (row.pane == "" && row.folder != "" && !row.session) {
		if cmd, dropped := m.dropOn(row.folder); dropped {
			return m, cmd
		}
		m.openFolder(row.folder)
		return m, nil
	}
	if row.session {
		return m, m.activateSession(row.key)
	}
	if row.pane != "" {
		if m.carrying() {
			m.clearHold()
			return m, m.showPane(row.pane)
		}
		return m.armPane(row.pane)
	}
	return m, nil
}

func (m *Model) activateRoot() tea.Cmd {
	if cmd, dropped := m.dropOn(""); dropped {
		return cmd
	}
	return m.showRoot()
}

func (m *Model) activateSession(key string) tea.Cmd {
	if m.sessionDir(key) && (m.held != "" || m.heldSession != "") {
		if cmd, dropped := m.dropOn(key); dropped {
			return cmd
		}
	}
	id := m.sessionFocus(key)
	if id != "" && m.armed == id && m.shown == key {
		m.clearHold()
		return m.showPane(id)
	}
	m.armed = id
	return m.selectSession(key)
}

func (m *Model) armPane(id string) (tea.Model, tea.Cmd) {
	if id == "" {
		return m, nil
	}
	if m.armed == id {
		m.armed = ""
		return m, m.showPane(id)
	}
	m.armed = id
	m.focus = id
	m.touch(id)
	m.drawer = true
	m.errText = ""
	return m, nil
}

func (m *Model) runSearchAction(action Action) (tea.Model, tea.Cmd) {
	m.find = ""
	m.armed = ""
	m.drawerAt = 0
	m.step = StepActions
	m.query = ""
	m.cursor = 0
	if actionNeedsPrompt(action) {
		m.drawer = false
		m.palette = true
	} else if action == ActionNew || action == ActionDetach || action == ActionUbersession {
		m.drawer = false
	}
	return m.applyAction(action)
}

func actionNeedsPrompt(action Action) bool {
	switch action {
	case ActionCommand, ActionSend, ActionFolder, ActionMove, ActionEnter, ActionFollow, ActionAlso, ActionUnfollow, ActionFocus, ActionKill:
		return true
	default:
		return false
	}
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
		if m.palette {
			m.drawer = false
		}
		return m, nil
	case hitToggle:
		return m, m.setDrawer(!m.drawer)
	case hitTab:
		return m, m.focusTab(item.id)
	case hitSession:
		return m, m.activateSession(item.id)
	case hitClose:
		return m, m.closeSession(item.id)
	case hitChoice:
		m.cursor = item.index
		return m.activate()
	case hitAction:
		rows := m.drawerRows()
		if item.index < 0 || item.index >= len(rows) || !rows[item.index].act {
			return m, nil
		}
		return m.runSearchAction(rows[item.index].action)
	case hitChild:
		if m.carrying() {
			m.clearHold()
			return m, m.showPane(item.id)
		}
		return m.armPane(item.id)
	case hitNewDir:
		m.startNaming()
		return m, nil
	case hitCarry:
		m.grabPane(item.id)
		return m, nil
	case hitCarrySession:
		m.grabSession(item.id)
		return m, nil
	case hitFolder:
		if cmd, dropped := m.dropOn(item.id); dropped {
			return m, cmd
		}
		m.openFolder(item.id)
		return m, nil
	case hitRoot:
		return m, m.activateRoot()
	default:
		return m, nil
	}
}

func (m *Model) carrying() bool {
	return m.held != "" || m.heldSession != "" || m.heldFolder != ""
}

func (m *Model) grabHighlight() {
	if m.drawerAt < 0 {
		return
	}
	rows := m.drawerRows()
	if m.drawerAt >= len(rows) {
		return
	}
	row := rows[m.drawerAt]
	switch {
	case row.pane != "":
		m.grabPane(row.pane)
	case row.session:
		m.grabSession(row.key)
	case row.folder != "":
		m.grabFolder(row.folder)
	}
}

func (m *Model) grabPane(id string) {
	if m.held == id {
		m.held = ""
		return
	}
	m.pickPane(id)
}

func (m *Model) grabSession(key string) {
	if m.heldSession == key {
		m.heldSession = ""
		return
	}
	m.pickSession(key)
}

func (m *Model) grabFolder(path string) {
	if m.heldFolder == path {
		m.heldFolder = ""
		return
	}
	m.pickFolder(path)
}

func (m *Model) clearHold() {
	m.held = ""
	m.heldSession = ""
	m.heldFolder = ""
}

func (m *Model) pickSession(key string) {
	m.heldSession = key
	m.held = ""
	m.heldFolder = ""
	m.drawer = true
	m.errText = ""
}

func (m *Model) pickFolder(path string) {
	m.heldFolder = path
	m.held = ""
	m.heldSession = ""
	m.drawer = true
	m.errText = ""
}

func (m *Model) pickPane(id string) {
	m.held = id
	m.heldSession = ""
	m.heldFolder = ""
	m.focus = id
	m.touch(id)
	m.drawer = true
	m.errText = ""
}

func (m *Model) dropOn(folder string) (tea.Cmd, bool) {
	switch {
	case m.held != "":
		return m.moveInto(folder), true
	case m.heldSession != "":
		if folder == m.heldSession {
			return nil, false
		}
		if folder != "" && strings.HasPrefix(folder, m.heldSession+"/") {
			m.errText = ErrPath.Error()
			return nil, true
		}
		return m.nestSession(folder), true
	case m.heldFolder != "":
		if folder == m.heldFolder {
			return nil, false
		}
		return m.nestFolder(folder), true
	default:
		return nil, false
	}
}

func (m *Model) showRoot() tea.Cmd {
	m.shown = ""
	m.followAll = true
	m.followPrefix = ""
	m.follow = nil
	m.drawer = true
	m.palette = false
	m.armed = ""
	m.errText = ""
	return m.rewatch()
}

func (m *Model) focusTab(id string) tea.Cmd {
	pane, ok := m.byID[id]
	if !ok {
		return nil
	}
	m.drawer = false
	m.palette = false
	m.find = ""
	m.armed = ""
	m.focus = id
	m.touch(id)
	m.errText = ""
	if m.followAll && m.shown == "" {
		return nil
	}
	key := m.sessionKey(pane)
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
	m.find = ""
	m.armed = ""
	m.errText = ""
	key := m.sessionKey(pane)
	m.shown = key
	if panePath(pane) != key {
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
	ids := m.panesUnder(key)
	m.made = withoutPrefix(m.made, key)
	if m.place == key || strings.HasPrefix(m.place, key+"/") {
		m.place = ""
	}
	if m.heldSession == key || strings.HasPrefix(m.heldSession, key+"/") {
		m.heldSession = ""
	}
	if m.heldFolder == key || strings.HasPrefix(m.heldFolder, key+"/") {
		m.heldFolder = ""
	}
	if len(ids) == 0 {
		if m.shown == key || strings.HasPrefix(m.shown, key+"/") {
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
	if m.shown == key || strings.HasPrefix(m.shown, key+"/") {
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

func (m *Model) sessionFocus(key string) string {
	panes := m.sessionPanes(key)
	if len(panes) == 0 {
		return ""
	}
	if contains(panes, m.focus) {
		return m.focus
	}
	return panes[0]
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
	seen := map[string]bool{}
	var cmds []tea.Cmd
	for _, group := range m.sessionGroups() {
		dir := m.sessionRoom(group.key)
		if dir == "" || seen[dir] {
			continue
		}
		seen[dir] = true
		cmds = append(cmds, m.readGitDir(dir))
	}
	return tea.Batch(cmds...)
}

func (m *Model) readGitDir(dir string) tea.Cmd {
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
	m.shown = path
	m.place = path
	m.naming = false
	m.name = ""
	m.errText = ""
	m.drawer = true
}

func (m *Model) moveInto(folder string) tea.Cmd {
	id := m.held
	if id == "" {
		return nil
	}
	pane, ok := m.byID[id]
	if !ok {
		m.held = ""
		return nil
	}
	base := pathBase(panePath(pane))
	next := base
	if folder != "" {
		next = folder + "/" + base
	}
	if panePath(pane) == next {
		m.clearHold()
		return nil
	}
	if m.keyTaken(next, id, false) {
		m.errText = ErrPath.Error()
		return nil
	}
	m.clearHold()
	if key, _, _ := m.sessionOf(next); key != "" {
		m.shown = key
	}
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

func (m *Model) sessionKey(pane Pane) string {
	key, _, _ := m.sessionOf(panePath(pane))
	if key == "" {
		return panePath(pane)
	}
	return key
}

func (m *Model) sessionOf(path string) (key, name, parent string) {
	if path == "" {
		return "", "", ""
	}
	parent = m.longestMadePrefix(path)
	rest := path
	if parent != "" {
		rest = strings.TrimPrefix(path, parent+"/")
	}
	name = sessionKeyFromPath(rest)
	if name == "" {
		return "", "", parent
	}
	key = name
	if parent != "" {
		key = parent + "/" + name
	}
	return key, name, parent
}

func (m *Model) longestMadePrefix(path string) string {
	best := ""
	for _, folder := range m.made {
		if folder == "" || folder == path {
			continue
		}
		if strings.HasPrefix(path, folder+"/") && len(folder) > len(best) {
			best = folder
		}
	}
	return best
}

func (m *Model) madeParent(path string) string {
	return m.longestMadePrefix(path)
}

func (m *Model) openFolder(path string) {
	if path == "" {
		return
	}
	m.armed = ""
	m.drawer = true
	if contains(m.made, path) {
		m.shown = path
		return
	}
	if key, _, _ := m.sessionOf(path); key != "" {
		m.shown = key
	}
}

func (m *Model) sessionRoom(key string) string {
	group, ok := m.group(key)
	if !ok {
		return ""
	}
	if contains(group.panes, m.focus) {
		if pane, ok := m.byID[m.focus]; ok && pane.Dir != "" {
			return pane.Dir
		}
	}
	for _, id := range group.panes {
		if pane, ok := m.byID[id]; ok && pane.Dir != "" {
			return pane.Dir
		}
	}
	return m.cwd
}

func (m *Model) gitStatus(dir string) string {
	if dir == "" || m.gitBy == nil {
		return ""
	}
	return m.gitBy[dir]
}

func (m *Model) panesUnder(key string) []string {
	var ids []string
	for _, id := range m.order {
		pane, ok := m.byID[id]
		if !ok {
			continue
		}
		path := panePath(pane)
		if path == key || strings.HasPrefix(path, key+"/") {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m *Model) nestSession(folder string) tea.Cmd {
	key := m.heldSession
	if key == "" {
		return nil
	}
	if folder != "" && (folder == key || strings.HasPrefix(folder, key+"/")) {
		m.errText = ErrPath.Error()
		return nil
	}
	group, ok := m.group(key)
	if !ok {
		m.heldSession = ""
		return nil
	}
	newKey := group.name
	if folder != "" {
		newKey = folder + "/" + group.name
	}
	if newKey == key {
		m.heldSession = ""
		return nil
	}
	if m.pathBusy(newKey, key) {
		m.errText = ErrPath.Error()
		return nil
	}
	type plan struct {
		id   string
		next string
	}
	var plans []plan
	for _, id := range group.panes {
		pane, ok := m.byID[id]
		if !ok {
			continue
		}
		old := panePath(pane)
		next := newKey + strings.TrimPrefix(old, key)
		if m.keyTaken(next, id, false) {
			m.errText = ErrPath.Error()
			return nil
		}
		plans = append(plans, plan{id: id, next: next})
	}
	for i, path := range m.made {
		if path == key || strings.HasPrefix(path, key+"/") {
			m.made[i] = newKey + strings.TrimPrefix(path, key)
		}
	}
	shown := newKey
	if len(plans) > 0 {
		if nextKey, _, _ := m.sessionOf(plans[0].next); nextKey != "" {
			shown = nextKey
		}
	}
	m.clearHold()
	m.place = folder
	m.shown = shown
	m.folder = rebasePath(m.folder, key, newKey)
	m.drawer = true
	m.palette = false
	m.errText = ""
	m.followAll = false
	m.followPrefix = shown
	m.follow = nil
	cmds := make([]tea.Cmd, 0, len(plans)+1)
	for _, item := range plans {
		cmds = append(cmds, m.move(item.id, item.next))
	}
	cmds = append(cmds, m.rewatch())
	return tea.Batch(cmds...)
}

func (m *Model) nestFolder(dest string) tea.Cmd {
	src := m.heldFolder
	if src == "" {
		return nil
	}
	base := pathBase(src)
	next := base
	if dest != "" {
		next = dest + "/" + base
	}
	if next == src {
		m.heldFolder = ""
		return nil
	}
	if dest != "" && (dest == src || strings.HasPrefix(dest, src+"/")) {
		m.errText = ErrPath.Error()
		return nil
	}
	if m.pathBusy(next, src) {
		m.errText = ErrPath.Error()
		return nil
	}
	type plan struct {
		id   string
		next string
	}
	var plans []plan
	for _, id := range m.order {
		pane, ok := m.byID[id]
		if !ok {
			continue
		}
		old := panePath(pane)
		moved := rebasePath(old, src, next)
		if moved == old {
			continue
		}
		if m.pathBusy(moved, src) {
			m.errText = ErrPath.Error()
			return nil
		}
		plans = append(plans, plan{id: id, next: moved})
	}
	for i, path := range m.made {
		m.made[i] = rebasePath(path, src, next)
	}
	m.shown = rebasePath(m.shown, src, next)
	m.folder = rebasePath(m.folder, src, next)
	m.followPrefix = rebasePath(m.followPrefix, src, next)
	m.clearHold()
	m.place = dest
	m.drawer = true
	m.palette = false
	m.errText = ""
	cmds := make([]tea.Cmd, 0, len(plans)+1)
	for _, item := range plans {
		cmds = append(cmds, m.move(item.id, item.next))
	}
	cmds = append(cmds, m.rewatch())
	return tea.Batch(cmds...)
}

// pathBusy reports a key outside src that already owns path.
// A folder that is only a parent of path is free.
func (m *Model) pathBusy(path, src string) bool {
	if path == "" {
		return false
	}
	under := func(have string) bool {
		return src != "" && (have == src || strings.HasPrefix(have, src+"/"))
	}
	for _, id := range m.order {
		pane, ok := m.byID[id]
		if !ok {
			continue
		}
		have := panePath(pane)
		if under(have) {
			continue
		}
		if have == path || strings.HasPrefix(path, have+"/") || strings.HasPrefix(have, path+"/") {
			return true
		}
	}
	for _, made := range m.made {
		if under(made) {
			continue
		}
		if made == path || strings.HasPrefix(made, path+"/") {
			return true
		}
	}
	return false
}

func rebasePath(path, src, next string) string {
	if src == "" || path == "" {
		return path
	}
	if path == src {
		return next
	}
	if strings.HasPrefix(path, src+"/") {
		return next + strings.TrimPrefix(path, src)
	}
	return path
}

func sessionKeyFromPath(path string) string {
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return path
}

func sessionLabel(head, cwd, git string, room int) string {
	if room <= 0 {
		return ""
	}
	tail := ""
	if git != "" {
		tail = "  " + git
	}
	if runewidth.StringWidth(head+tail) > room {
		return trimWidth(head+tail, room)
	}
	if cwd == "" {
		return head + tail
	}
	budget := room - runewidth.StringWidth(head) - runewidth.StringWidth(tail) - 2
	if budget < 1 {
		return head + tail
	}
	return head + "  " + trimWidth(cwd, budget) + tail
}

func trimWidth(text string, width int) string {
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
	return b.String()
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

func dropID(ids []string, id string) []string {
	out := make([]string, 0, len(ids))
	for _, item := range ids {
		if item != id {
			out = append(out, item)
		}
	}
	return out
}
