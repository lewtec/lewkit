package term

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Session is the ubersession database.
// Pane records live here. Which client is looking at which pane does not.
type Session struct {
	ctx    context.Context
	cancel context.CancelFunc

	mu         sync.Mutex
	id         string
	panes      map[string]*live
	paths      map[string]string
	order      []string
	listeners  map[int]*listener
	nextListen int
	clients    int
	served     bool
	idle       func()
	closed     bool
}

type live struct {
	info PaneInfo
	grid *Grid
	proc *process
}

type listener struct {
	all    bool
	prefix string
	panes  map[string]bool
	fn     func(Event)
}

// Open starts an empty ubersession.
// Cancelling ctx closes the session and its pane processes.
func Open(ctx context.Context) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, err := randomID(8)
	if err != nil {
		return nil, err
	}
	child, cancel := context.WithCancel(ctx)
	session := &Session{
		ctx:       child,
		cancel:    cancel,
		id:        id,
		panes:     map[string]*live{},
		paths:     map[string]string{},
		listeners: map[int]*listener{},
	}
	go func() {
		<-ctx.Done()
		session.Close()
	}()
	return session, nil
}

// Close stops every pane. It is safe to call more than once.
func (s *Session) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	procs := make([]*process, 0, len(s.panes))
	for _, pane := range s.panes {
		pane.info.Alive = false
		if pane.proc != nil {
			procs = append(procs, pane.proc)
			pane.proc = nil
		}
	}
	cancel := s.cancel
	s.mu.Unlock()
	for _, proc := range procs {
		proc.Close()
	}
	if cancel != nil {
		cancel()
	}
}

// stopWhenIdle asks Listen to return once a client has connected and then
// every client is gone and no panes remain. Panes keep the process up after
// detach, the same way a tmux server outlives its clients.
func (s *Session) stopWhenIdle(stop func()) {
	s.mu.Lock()
	s.idle = stop
	s.mu.Unlock()
}

func (s *Session) clientArrived() {
	s.mu.Lock()
	s.clients++
	s.served = true
	s.mu.Unlock()
}

func (s *Session) clientLeft() {
	s.mu.Lock()
	if s.clients > 0 {
		s.clients--
	}
	stop := s.idle
	idle := stop != nil && s.served && s.clients == 0 && len(s.panes) == 0
	s.mu.Unlock()
	if idle {
		stop()
	}
}

// Info returns the membership record.
func (s *Session) Info(ctx context.Context) (SessionInfo, error) {
	if err := ctx.Err(); err != nil {
		return SessionInfo{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return SessionInfo{}, ErrTermClosed
	}
	return s.info(), nil
}

func (s *Session) info() SessionInfo {
	panes := append([]string(nil), s.order...)
	return SessionInfo{ID: s.id, Panes: panes, Entries: s.entries()}
}

func (s *Session) entries() []Entry {
	return catalog(s.order, s.panes)
}

// Panes returns every pane header in creation order.
func (s *Session) Panes(ctx context.Context) ([]PaneInfo, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrTermClosed
	}
	out := make([]PaneInfo, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.panes[id].info)
	}
	return out, nil
}

// Pane returns one record, including the visible grid.
func (s *Session) Pane(ctx context.Context, id string) (Pane, error) {
	if err := ctx.Err(); err != nil {
		return Pane{}, err
	}
	if id == "" {
		return Pane{}, ErrID
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Pane{}, ErrTermClosed
	}
	pane, ok := s.panes[id]
	if !ok {
		return Pane{}, fmt.Errorf("%w: %s", ErrPane, id)
	}
	return pane.record(), nil
}

// Create starts a process and stores its pane.
// The initial size is not a write: LastWriter stays empty.
func (s *Session) Create(ctx context.Context, req CreateRequest) (Pane, error) {
	if err := ctx.Err(); err != nil {
		return Pane{}, err
	}
	rows, cols := req.Rows, req.Cols
	if rows == 0 && cols == 0 {
		rows, cols = 24, 80
	}
	if rows < 1 || cols < 1 {
		return Pane{}, ErrSize
	}
	command := req.Command
	if len(command) == 0 {
		command = shellCommand()
	}
	command = append([]string(nil), command...)
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return Pane{}, ErrTermClosed
	}
	id, err := s.freshID()
	if err != nil {
		s.mu.Unlock()
		return Pane{}, err
	}
	path, err := assignPath(req.Path, id)
	if err != nil {
		s.mu.Unlock()
		return Pane{}, err
	}
	if pathTaken(s.paths, path) {
		s.mu.Unlock()
		return Pane{}, fmt.Errorf("%w: %s", ErrPath, path)
	}
	procCtx := s.ctx
	s.mu.Unlock()

	proc, err := startProcess(procCtx, command, req.Dir, rows, cols)
	if err != nil {
		return Pane{}, err
	}
	title := req.Title
	if title == "" {
		title = filepath.Base(command[0])
	}
	pane := &live{
		info: PaneInfo{
			ID:       id,
			Title:    title,
			Command:  command,
			Dir:      req.Dir,
			Alive:    true,
			Exit:     -1,
			Rows:     rows,
			Cols:     cols,
			Revision: 1,
			Path:     path,
		},
		grid: NewGrid(rows, cols),
		proc: proc,
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		proc.Close()
		return Pane{}, ErrTermClosed
	}
	if pathTaken(s.paths, path) {
		s.mu.Unlock()
		proc.Close()
		return Pane{}, fmt.Errorf("%w: %s", ErrPath, path)
	}
	s.panes[id] = pane
	s.paths[path] = id
	s.order = append(s.order, id)
	record := pane.record()
	info := s.info()
	fns := s.matching(Event{Notice: NoticePane, Pane: &record})
	fns = append(fns, s.matching(Event{Notice: NoticeSession, Session: &info})...)
	s.mu.Unlock()
	go s.readLoop(id, proc)
	deliver(fns)
	return record, nil
}

// Write sends input to a pane.
// A non-empty Data with a positive size records that client as the last writer
// and reflows when the size differs. Data without a size does not resize.
func (s *Session) Write(ctx context.Context, req WriteRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if req.ID == "" {
		return ErrID
	}
	if (req.Rows > 0) != (req.Cols > 0) || req.Rows < 0 || req.Cols < 0 {
		return ErrSize
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrTermClosed
	}
	pane, ok := s.panes[req.ID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrPane, req.ID)
	}
	if !pane.info.Alive || pane.proc == nil {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrTermClosed, req.ID)
	}
	proc := pane.proc
	var fns []func()
	if req.Data != "" && req.Rows > 0 && req.Cols > 0 {
		pane.info.LastWriter = req.Client
		if req.Rows != pane.info.Rows || req.Cols != pane.info.Cols {
			pane.grid.Reflow(req.Rows, req.Cols)
			pane.info.Rows = req.Rows
			pane.info.Cols = req.Cols
			pane.info.Revision++
			syncCursor(&pane.info, pane.grid)
			record := pane.record()
			fns = s.matching(Event{Notice: NoticePane, Pane: &record})
			if err := proc.Resize(req.Rows, req.Cols); err != nil {
				s.mu.Unlock()
				deliver(fns)
				return err
			}
		}
	}
	s.mu.Unlock()
	deliver(fns)
	if req.Data == "" {
		return nil
	}
	return proc.Write([]byte(req.Data))
}

// List returns the files and folders directly inside path.
// An empty path is the root. A path that names a file is ErrPath.
// A folder with no panes is an empty list. The server does not store empty folders.
func (s *Session) List(ctx context.Context, folder string) (Tree, error) {
	if err := ctx.Err(); err != nil {
		return Tree{}, err
	}
	cleaned, err := cleanPath(folder)
	if err != nil {
		return Tree{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Tree{}, ErrTermClosed
	}
	if cleaned != "" {
		if _, file := s.paths[cleaned]; file {
			return Tree{}, fmt.Errorf("%w: %s", ErrPath, cleaned)
		}
	}
	return Tree{Path: cleaned, Entries: directChildren(s.entries(), cleaned)}, nil
}

// Move gives a pane a new key.
// The key must not collide with a file or with a folder that already holds panes.
func (s *Session) Move(ctx context.Context, id, path string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return ErrID
	}
	cleaned, err := cleanPath(path)
	if err != nil {
		return err
	}
	if cleaned == "" {
		return fmt.Errorf("%w: %s", ErrPath, path)
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrTermClosed
	}
	pane, ok := s.panes[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrPane, id)
	}
	old := pane.info.Path
	if old == cleaned {
		s.mu.Unlock()
		return nil
	}
	delete(s.paths, old)
	if pathTaken(s.paths, cleaned) {
		s.paths[old] = id
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrPath, cleaned)
	}
	pane.info.Path = cleaned
	pane.info.Revision++
	s.paths[cleaned] = id
	record := pane.record()
	info := s.info()
	fns := s.matching(Event{Notice: NoticePane, Pane: &record})
	fns = append(fns, s.matching(Event{Notice: NoticeSession, Session: &info})...)
	s.mu.Unlock()
	deliver(fns)
	return nil
}

// Kill stops the pane process and keeps the grid.
func (s *Session) Kill(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return ErrID
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrTermClosed
	}
	pane, ok := s.panes[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrPane, id)
	}
	proc := pane.proc
	pane.proc = nil
	s.mu.Unlock()
	if proc != nil {
		proc.Close()
	}
	return nil
}

// Watch registers fn for matching pushes.
// An empty panes list with all watches the ubersession. prefix watches a folder.
// The cancel function removes the registration. fn runs without the session lock.
func (s *Session) Watch(panes []string, all bool, prefix string, fn func(Event)) (func(), error) {
	if fn == nil {
		return nil, ErrProtocol
	}
	if prefix != "" {
		cleaned, err := cleanPath(prefix)
		if err != nil {
			return nil, err
		}
		prefix = cleaned
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrTermClosed
	}
	s.nextListen++
	id := s.nextListen
	ln := &listener{all: all, prefix: prefix, panes: map[string]bool{}, fn: fn}
	if !all {
		for _, pane := range panes {
			if pane != "" {
				ln.panes[pane] = true
			}
		}
	}
	s.listeners[id] = ln
	initial := s.initial(ln)
	s.mu.Unlock()
	for _, ev := range initial {
		fn(ev)
	}
	return func() {
		s.mu.Lock()
		delete(s.listeners, id)
		s.mu.Unlock()
	}, nil
}

func (s *Session) initial(ln *listener) []Event {
	var out []Event
	if ln.all || ln.prefix != "" {
		info := s.info()
		out = append(out, Event{Notice: NoticeSession, Session: &info})
	}
	if ln.all {
		for _, id := range s.order {
			record := s.panes[id].record()
			out = append(out, Event{Notice: NoticePane, Pane: &record})
		}
		return out
	}
	for _, id := range s.order {
		pane := s.panes[id]
		if ln.panes[id] || (ln.prefix != "" && pathUnder(pane.info.Path, ln.prefix)) {
			record := pane.record()
			out = append(out, Event{Notice: NoticePane, Pane: &record})
		}
	}
	return out
}

func (s *Session) readLoop(id string, proc *process) {
	buf := make([]byte, 4096)
	for {
		n, err := proc.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			s.feed(id, chunk)
		}
		if err != nil {
			s.exited(id, proc.finish())
			return
		}
	}
}

func (s *Session) feed(id string, chunk []byte) {
	s.mu.Lock()
	pane, ok := s.panes[id]
	if !ok || s.closed {
		s.mu.Unlock()
		return
	}
	pane.grid.Feed(chunk)
	pane.info.Revision++
	syncCursor(&pane.info, pane.grid)
	record := pane.record()
	fns := s.matching(Event{Notice: NoticePane, Pane: &record})
	s.mu.Unlock()
	deliver(fns)
}

func (s *Session) exited(id string, code int) {
	s.mu.Lock()
	pane, ok := s.panes[id]
	if !ok {
		s.mu.Unlock()
		return
	}
	pane.info.Alive = false
	pane.info.Exit = code
	pane.info.Revision++
	pane.proc = nil
	record := pane.record()
	info := s.info()
	fns := s.matching(Event{Notice: NoticePane, Pane: &record})
	fns = append(fns, s.matching(Event{Notice: NoticePaneClosed, Pane: &record})...)
	fns = append(fns, s.matching(Event{Notice: NoticeSession, Session: &info})...)
	s.mu.Unlock()
	deliver(fns)
}

func (s *Session) matching(ev Event) []func() {
	var fns []func()
	for _, ln := range s.listeners {
		if !ln.match(ev) {
			continue
		}
		copied := ev
		if ev.Pane != nil {
			pane := *ev.Pane
			copied.Pane = &pane
		}
		if ev.Session != nil {
			info := *ev.Session
			info.Panes = append([]string(nil), ev.Session.Panes...)
			info.Entries = append([]Entry(nil), ev.Session.Entries...)
			copied.Session = &info
		}
		event := copied
		fns = append(fns, func() { ln.fn(event) })
	}
	return fns
}

func (ln *listener) match(ev Event) bool {
	switch ev.Notice {
	case NoticeSession:
		return ln.all || ln.prefix != ""
	case NoticePane, NoticePaneClosed:
		if ln.all {
			return true
		}
		if ev.Pane == nil {
			return false
		}
		if ln.panes[ev.Pane.ID] {
			return true
		}
		return ln.prefix != "" && pathUnder(ev.Pane.Path, ln.prefix)
	default:
		return false
	}
}

func deliver(fns []func()) {
	for _, fn := range fns {
		fn()
	}
}

func (s *Session) freshID() (string, error) {
	for range 8 {
		id, err := randomID(4)
		if err != nil {
			return "", err
		}
		if _, ok := s.panes[id]; !ok {
			return id, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrID, "no free pane id")
}

func (p *live) record() Pane {
	rows, cols, cursorRow, cursorCol, cells := p.grid.Snapshot()
	info := p.info
	info.Rows = rows
	info.Cols = cols
	info.CursorRow = cursorRow
	info.CursorCol = cursorCol
	return Pane{PaneInfo: info, Cells: cells}
}

func syncCursor(info *PaneInfo, grid *Grid) {
	info.CursorRow, info.CursorCol = grid.Cursor()
}

func shellCommand() []string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return []string{shell}
}

func randomID(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("%w: %w", ErrSocket, err)
	}
	return hex.EncodeToString(buf), nil
}
