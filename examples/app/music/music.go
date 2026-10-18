package main

import (
	"context"
	"errors"
	"image"
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/launcher"
	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/image/convert/file"
	"github.com/lewtec/lewkit/x/ui/gui"
)

type chooseFunc func(context.Context, filedialog.Request) ([]string, error)

type promptFunc func(context.Context, string) (string, error)

type searchAsked struct {
	text string
	err  error
}

type openFunc func(...string) (fs.FS, error)

// openMusic builds the browser. The returned func closes the catalog.
// dir is ingested on the first tick. An empty dir waits for a drop.
func openMusic(ctx context.Context, dir string) (gui.Model, func() error, error) {
	lib, err := OpenLibrary(ctx)
	if err != nil {
		return nil, nil, err
	}
	play := newPlayer(nil)
	play.openFile = lib.Open
	go play.loop(ctx)
	model := newMusic(ctx, lib, play)
	model.dir = dir
	return model, lib.Close, nil
}

type musicScreen int

const (
	musicBrowse musicScreen = iota
	musicAlbum
	musicNow
)

type ingested struct {
	count int
	err   error
}

type chosen struct {
	paths []string
	err   error
}

type musicModel struct {
	gui.Dirty
	gui.Metrics
	ctx      context.Context
	lib      *Library
	player   *player
	chooseFn chooseFunc
	promptFn promptFunc
	openFn   openFunc
	picking  bool
	size     image.Point
	dir      string
	query    string
	search   bool
	screen   musicScreen
	album    string
	albums   []Album
	tracks   []Track
	now      *Track
	resume   int64
	scroll   float32
	cardX    float32
	busy     bool
	booted   bool
	note     string
	mode     daynight.Mode
	paint    musicPaint
	covers   map[string]image.Image

	head    *headerModel
	shelf   *albumModel
	drag    gui.Drag
	dragKey string
	list    *trackModel
	stage   *nowModel
}

func newMusic(ctx context.Context, lib *Library, play *player) *musicModel {
	if play != nil && lib != nil {
		play.openFile = lib.Open
	}
	model := &musicModel{
		ctx:      ctx,
		lib:      lib,
		player:   play,
		chooseFn: filedialog.Choose,
		openFn:   filedialog.Open,
		size:     image.Pt(900, 700),
		mode:     daynight.Dark,
		head:     &headerModel{},
		shelf:    &albumModel{},
		list:     &trackModel{},
		stage:    &nowModel{},
	}
	model.prepare()
	return model
}

func (m *musicModel) Init() gui.Cmd { return gui.Tick() }

func (m *musicModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if m == nil {
		return m, nil
	}
	m.Observe(msg)
	defer m.prepare()
	if size, ok := guiSize(msg); ok && size.X > 0 && size.Y > 0 {
		m.Dirty, m.size = gui.See(m.Dirty, m.size, size)
	}
	if tick, ok := msg.(gui.TickMsg); ok {
		if m.dir != "" && !m.booted {
			m.booted = true
			m.Dirty = gui.Touch(m.Dirty)
			return m, m.ingest([]string{m.dir})
		}
		if m.player != nil && m.player.Playing() {
			m.Dirty, m.resume = gui.See(m.Dirty, m.resume, m.player.Played())
			return m, gui.Every(tick.Period)
		}
		return m, nil
	}
	switch event := msg.(type) {
	case gui.ModeMsg:
		m.Dirty, m.mode = gui.See(m.Dirty, m.mode, event.Mode)
	case pickedAlbum:
		m.Dirty, m.album = gui.See(m.Dirty, m.album, event.name)
		m.Dirty, m.screen = gui.See(m.Dirty, m.screen, musicAlbum)
		m.list.scroll = 0
		m.reload()
		return m, nil
	case pickedTrack:
		m.start(event.track)
		m.Dirty = gui.Touch(m.Dirty)
		return m, gui.Tick()
	case pickedOpen:
		return m, m.choose()
	case pickedBack:
		m.Dirty, m.screen = gui.See(m.Dirty, m.screen, musicBrowse)
		m.list.scroll = 0
		m.reload()
		return m, nil
	case pickedQuery:
		m.Dirty, m.query = gui.See(m.Dirty, m.query, event.text)
		m.reload()
		return m, nil
	case searchAsked:
		return m.onSearchAsked(event)
	case pickedPlay:
		m.toggle()
		m.Dirty = gui.Touch(m.Dirty)
		if m.player != nil && m.player.Playing() {
			return m, gui.Tick()
		}
		return m, nil
	case pickedStep:
		m.step(event.delta)
		m.Dirty = gui.Touch(m.Dirty)
		return m, nil
	case pickedSeek:
		if m.now != nil && m.player != nil {
			m.Dirty, m.resume = gui.See(m.Dirty, m.resume, event.frame)
			m.player.Play(m.now.Path, event.frame)
		}
		return m, nil
	case chosen:
		m.picking = false
		if event.err != nil {
			m.note = event.err.Error()
			m.Dirty = gui.Touch(m.Dirty)
			return m, nil
		}
		if len(event.paths) == 0 {
			return m, nil
		}
		return m, m.ingest(event.paths)
	case ingested:
		m.busy = false
		if event.err != nil {
			m.note = event.err.Error()
		} else if event.count == 0 {
			m.note = "no audio in that folder"
		} else {
			m.note = ""
		}
		m.Dirty = gui.Touch(m.Dirty)
		m.reload()
		if m.player != nil && m.player.Playing() {
			return m, gui.Tick()
		}
		return m, nil
	case window.Drop:
		return m, m.ingest(event.Paths)
	case window.Key:
		return m.delegate(event)
	case window.Pointer:
		return m.aim(event)
	case window.Scroll:
		return m.wheel(event)
	}
	return m, nil
}

func guiSize(msg gui.Msg) (image.Point, bool) {
	switch message := msg.(type) {
	case gui.TickMsg:
		return message.Size, message.Size.X > 0
	case window.Resize:
		return message.Size, true
	default:
		return image.Point{}, false
	}
}

func (m *musicModel) hit(pos image.Point) (string, float32, float32) {
	root := m.View()
	if root == nil {
		return "", 0, 0
	}
	size := gui.Size{Width: float32(m.size.X), Height: float32(m.size.Y)}
	return m.Hit(root, size, pos)
}

func (m *musicModel) ingest(paths []string) gui.Cmd {
	m.busy = true
	m.note = "reading library"
	m.Dirty = gui.Touch(m.Dirty)
	lib := m.lib
	open := m.openFn
	return func(ctx context.Context) gui.Msg {
		count, err := lib.IngestPaths(ctx, open, paths...)
		return ingested{count: count, err: err}
	}
}

func (m *musicModel) reload() {
	if m.lib == nil {
		return
	}
	album := ""
	if m.screen == musicAlbum {
		album = m.album
	}
	albums, err := m.lib.Albums(m.ctx, m.query)
	if err != nil {
		m.note = err.Error()
		return
	}
	tracks, err := m.lib.Tracks(m.ctx, album, m.query)
	if err != nil {
		m.note = err.Error()
		return
	}
	m.albums = albums
	m.tracks = tracks
	m.Dirty = gui.Touch(m.Dirty)
}

func (m *musicModel) aim(event window.Pointer) (gui.Model, gui.Cmd) {
	if m == nil {
		return m, nil
	}
	if event.Button == 1 && event.Pressed {
		m.dragKey, _, _ = m.hit(event.Pos)
	}
	step, ok := m.drag.Step(event, gui.Slop(m.Scale))
	if !ok {
		return m, nil
	}
	if !step.Tap {
		dx, dy := float32(step.Delta.X), float32(step.Delta.Y)
		// The music tree is laid out in device pixels, so the finger
		// distance is the scroll distance. A sideways drag on the album
		// row moves that row. Anything else moves the track list.
		if strings.HasPrefix(m.dragKey, "album") && abs32(dx) >= abs32(dy) {
			m.nudgeShelf(dx)
			return m, nil
		}
		m.nudgeList(dy)
		return m, nil
	}
	event.Pos = step.At
	event.Button = 1
	event.Pressed = true
	key, along, _ := m.hit(event.Pos)
	if key != "search" {
		m.Dirty, m.head.search = gui.See(m.Dirty, m.head.search, false)
	}
	switch {
	case key == "open":
		return m, m.choose()
	case key == "search":
		return m, m.askSearch()
	case key == "back":
		m.Dirty, m.screen = gui.See(m.Dirty, m.screen, musicBrowse)
		m.list.scroll = 0
		m.reload()
		return m, nil
	case strings.HasPrefix(key, "album:"):
		m.Dirty, m.album = gui.See(m.Dirty, m.album, strings.TrimPrefix(key, "album:"))
		m.Dirty, m.screen = gui.See(m.Dirty, m.screen, musicAlbum)
		m.list.scroll = 0
		m.reload()
		return m, nil
	case strings.HasPrefix(key, "track:"):
		if track, ok := m.trackID(key); ok {
			m.start(track)
			m.Dirty = gui.Touch(m.Dirty)
			return m, gui.Tick()
		}
	case key == "play":
		m.toggle()
		m.Dirty = gui.Touch(m.Dirty)
		if m.player != nil && m.player.Playing() {
			return m, gui.Tick()
		}
	case key == "prev":
		m.step(-1)
		m.Dirty = gui.Touch(m.Dirty)
	case key == "next":
		m.step(1)
		m.Dirty = gui.Touch(m.Dirty)
	case key == "bar":
		if m.now != nil && m.player != nil && m.player.Total() > 0 {
			if along < 0 {
				along = 0
			}
			if along > 1 {
				along = 1
			}
			frame := int64(along * float32(m.player.Total()))
			m.Dirty, m.resume = gui.See(m.Dirty, m.resume, frame)
			m.player.Play(m.now.Path, frame)
		}
	}
	return m, nil
}

func (m *musicModel) wheel(event window.Scroll) (gui.Model, gui.Cmd) {
	if m == nil {
		return m, nil
	}
	key, _, _ := m.hit(event.Pos)
	switch {
	case strings.HasPrefix(key, "album"):
		m.nudgeShelf(float32(event.Delta.Y + event.Delta.X))
	case strings.HasPrefix(key, "track"):
		m.nudgeList(float32(event.Delta.Y))
	}
	return m, nil
}

func (m *musicModel) nudgeShelf(delta float32) {
	if m == nil || m.shelf == nil || delta == 0 {
		return
	}
	next := m.shelf.scroll + delta
	if next < 0 {
		next = 0
	}
	m.shelf.Dirty, m.shelf.scroll = gui.See(m.shelf.Dirty, m.shelf.scroll, next)
	m.absorb(m.shelf)
}

func (m *musicModel) nudgeList(delta float32) {
	if m == nil || m.list == nil || delta == 0 {
		return
	}
	next := m.list.scroll + delta
	if next < 0 {
		next = 0
	}
	if limit := m.list.maxScroll(); next > limit {
		next = limit
	}
	m.list.Dirty, m.list.scroll = gui.See(m.list.Dirty, m.list.scroll, next)
	m.absorb(m.list)
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func (m *musicModel) trackID(key string) (Track, bool) {
	id, err := strconv.ParseInt(strings.TrimPrefix(key, "track:"), 10, 64)
	if err != nil {
		return Track{}, false
	}
	for _, track := range m.tracks {
		if track.ID == id {
			return track, true
		}
	}
	return Track{}, false
}

func (m *musicModel) delegate(msg gui.Msg) (gui.Model, gui.Cmd) {
	m.prepare()
	cmd := take(&m.head, msg)
	if cmd == nil {
		switch m.screen {
		case musicNow:
			cmd = take(&m.stage, msg)
		case musicAlbum:
			cmd = take(&m.list, msg)
		default:
			cmd = take(&m.shelf, msg)
			if cmd == nil {
				cmd = take(&m.list, msg)
			}
		}
	}
	m.absorb(m.head, m.shelf, m.list, m.stage)
	return m, cmd
}

func (m *musicModel) absorb(parts ...interface{ Consume() bool }) {
	for _, part := range parts {
		if part != nil && part.Consume() {
			m.Dirty = gui.Touch(m.Dirty)
		}
	}
}

func take[T gui.Model](model *T, msg gui.Msg) gui.Cmd {
	if model == nil {
		return nil
	}
	current := gui.Model(*model)
	if current == nil {
		return nil
	}
	next, cmd := current.Update(msg)
	*model = next.(T)
	return cmd
}

func (m *musicModel) askSearch() gui.Cmd {
	ask := m.promptFn
	if ask == nil {
		ask = launcher.Prompt
	}
	return func(ctx context.Context) gui.Msg {
		text, err := ask(ctx, "Search")
		waitUI(ctx)
		return searchAsked{text: text, err: err}
	}
}

// waitUI returns after the UI thread has finished the dialog click and the
// window has settled. A frame during that resize faults the Android driver.
// A process with no UI thread, including a unit test, returns immediately.
func waitUI(ctx context.Context) {
	bound, on := thread.State()
	if !bound || on || ctx.Err() != nil {
		return
	}
	thread.Do(func() {})
	timer := time.NewTimer(400 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}

func (m *musicModel) onSearchAsked(event searchAsked) (gui.Model, gui.Cmd) {
	if event.err != nil {
		if errors.Is(event.err, errs.ErrCanceled) || errors.Is(event.err, context.Canceled) {
			return m, nil
		}
		m.note = event.err.Error()
		m.Dirty = gui.Touch(m.Dirty)
		return m, nil
	}
	text := strings.TrimSpace(event.text)
	if m.head != nil {
		m.head.query = text
		m.Dirty, m.head.search = gui.See(m.Dirty, m.head.search, false)
	}
	m.Dirty, m.query = gui.See(m.Dirty, m.query, text)
	m.reload()
	return m, nil
}

func (m *musicModel) choose() gui.Cmd {
	if m.picking {
		return nil
	}
	m.picking = true
	choose := m.chooseFn
	if choose == nil {
		choose = filedialog.Choose
	}
	return func(ctx context.Context) gui.Msg {
		paths, err := choose(ctx, filedialog.Request{Title: "Music folder", Folder: true})
		if errors.Is(err, filedialog.ErrCanceled) {
			return chosen{}
		}
		return chosen{paths: paths, err: err}
	}
}

func (m *musicModel) start(track Track) {
	m.now = &track
	m.resume = 0
	m.screen = musicNow
	if m.player != nil {
		m.player.Play(track.Path, 0)
	}
}

func (m *musicModel) toggle() {
	if m.now == nil || m.player == nil {
		return
	}
	if m.player.Playing() && m.player.Path() == m.now.Path {
		m.resume = m.player.Played()
		m.player.Stop()
		return
	}
	frame := m.resume
	if m.player.Total() > 0 && frame >= m.player.Total() {
		frame = 0
	}
	m.player.Play(m.now.Path, frame)
}

func (m *musicModel) step(delta int) {
	if m.now == nil {
		return
	}
	for i, track := range m.tracks {
		if track.ID != m.now.ID {
			continue
		}
		j := i + delta
		if j >= 0 && j < len(m.tracks) {
			m.start(m.tracks[j])
		}
		return
	}
}

func (m *musicModel) cover(path string) image.Image {
	if path == "" {
		return nil
	}
	if m.covers == nil {
		m.covers = map[string]image.Image{}
	}
	if img, ok := m.covers[path]; ok {
		return img
	}
	raw, err := m.readCoverBytes(path)
	if err != nil {
		m.covers[path] = nil
		return nil
	}
	img, err := file.Decode(raw)
	if err != nil {
		m.covers[path] = nil
		return nil
	}
	m.covers[path] = img
	return img
}

func (m *musicModel) readCoverBytes(path string) ([]byte, error) {
	if m.lib != nil {
		file, err := m.lib.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		return io.ReadAll(file)
	}
	return os.ReadFile(path)
}
