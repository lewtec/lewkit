package experiments

import (
	"context"
	"errors"
	"image"
	"os"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/image/convert"
	"github.com/lewtec/lewkit/x/ui/gui"
)

type musicCmd struct {
	dir    cmd.StringArg   `long:"dir" default:"" help:"music folder; empty waits for a drop"`
	width  cmd.IntArg[int] `long:"width" default:"900" help:"window width"`
	height cmd.IntArg[int] `long:"height" default:"700" help:"window height"`
}

func (musicCmd) Description() string { return "browse a dropped music folder" }

func (c *musicCmd) Run(ctx context.Context) error {
	lib, err := OpenLibrary(ctx)
	if err != nil {
		return err
	}
	defer lib.Close()
	play := newPlayer(nil)
	go play.loop(ctx)
	model := newMusic(ctx, lib, play)
	model.dir = c.dir.Value()
	return runGUI(ctx, "music", gui.Options{
		Title:  "lewkit music",
		Width:  c.width.Value(),
		Height: c.height.Value(),
	}, model)
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
	ctx    context.Context
	lib    *Library
	player *player
	size   image.Point
	dir    string
	query  string
	search bool
	screen musicScreen
	album  string
	albums []Album
	tracks []Track
	now    *Track
	resume int64
	scroll float32
	cardX  float32
	busy   bool
	booted bool
	note   string
	mode   daynight.Mode
	paint  musicPaint
	covers map[string]image.Image

	head  *headerModel
	shelf *albumModel
	list  *trackModel
	stage *nowModel
}

func newMusic(ctx context.Context, lib *Library, play *player) *musicModel {
	model := &musicModel{
		ctx:    ctx,
		lib:    lib,
		player: play,
		size:   image.Pt(900, 700),
		mode:   daynight.Dark,
		head:   &headerModel{},
		shelf:  &albumModel{},
		list:   &trackModel{},
		stage:  &nowModel{},
	}
	model.prepare()
	return model
}

func (m *musicModel) Init() gui.Cmd { return gui.Tick() }

func (m *musicModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if m == nil {
		return m, nil
	}
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

func (m *musicModel) ingest(paths []string) gui.Cmd {
	m.busy = true
	m.note = "reading library"
	m.Dirty = gui.Touch(m.Dirty)
	lib := m.lib
	return func() gui.Msg {
		var count int
		var failed error
		for _, path := range paths {
			if path == "" {
				continue
			}
			n, err := lib.Ingest(m.ctx, path)
			count += n
			if err != nil && n == 0 {
				failed = err
			}
		}
		if count > 0 {
			failed = nil
		}
		return ingested{count: count, err: failed}
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
	if m == nil || event.Button != 1 || !event.Pressed {
		return m, nil
	}
	key, along, _ := gui.Hit(m.View(), gui.Size{Width: float32(m.size.X), Height: float32(m.size.Y)}, event.Pos)
	if key != "search" {
		m.Dirty, m.head.search = gui.See(m.Dirty, m.head.search, false)
	}
	switch {
	case key == "open":
		return m, m.choose()
	case key == "search":
		m.Dirty, m.head.search = gui.See(m.Dirty, m.head.search, true)
		return m, nil
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
	key, _, _ := gui.Hit(m.View(), gui.Size{Width: float32(m.size.X), Height: float32(m.size.Y)}, event.Pos)
	switch {
	case strings.HasPrefix(key, "album"):
		next := m.shelf.scroll + float32(event.Delta.Y+event.Delta.X)
		if next < 0 {
			next = 0
		}
		m.shelf.Dirty, m.shelf.scroll = gui.See(m.shelf.Dirty, m.shelf.scroll, next)
		m.absorb(m.shelf)
	case strings.HasPrefix(key, "track"):
		next := m.list.scroll + float32(event.Delta.Y)
		if next < 0 {
			next = 0
		}
		if limit := m.list.maxScroll(); next > limit {
			next = limit
		}
		m.list.Dirty, m.list.scroll = gui.See(m.list.Dirty, m.list.scroll, next)
		m.absorb(m.list)
	}
	return m, nil
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

func (m *musicModel) choose() gui.Cmd {
	ctx := m.ctx
	return func() gui.Msg {
		paths, err := filedialog.Choose(ctx, filedialog.Request{Title: "Music folder", Folder: true})
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
	raw, err := os.ReadFile(path)
	if err != nil {
		m.covers[path] = nil
		return nil
	}
	img, err := convert.Decode(raw)
	if err != nil {
		m.covers[path] = nil
		return nil
	}
	m.covers[path] = img
	return img
}
