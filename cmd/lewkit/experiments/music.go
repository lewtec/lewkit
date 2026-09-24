package experiments

import (
	"context"
	"errors"
	"image"
	"os"

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

type musicHit struct {
	box   *gui.Box
	track *Track
	album string
}

func newMusic(ctx context.Context, lib *Library, play *player) *musicModel {
	return &musicModel{
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
}

func (m *musicModel) Init() gui.Cmd { return gui.Tick() }

func (m *musicModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if m == nil {
		return m, nil
	}
	if size, ok := guiSize(msg); ok && size.X > 0 && size.Y > 0 {
		m.size = size
	}
	if tick, ok := msg.(gui.TickMsg); ok {
		if m.dir != "" && !m.booted {
			m.booted = true
			return m, m.ingest([]string{m.dir})
		}
		if m.player != nil && m.player.Playing() {
			m.resume = m.player.Played()
		}
		return m, gui.Every(tick.Period)
	}
	switch event := msg.(type) {
	case gui.ModeMsg:
		m.mode = event.Mode
	case pickedAlbum:
		m.album = event.name
		m.screen = musicAlbum
		m.list.scroll = 0
		m.reload()
		return m, nil
	case pickedTrack:
		m.start(event.track)
		return m, nil
	case pickedOpen:
		return m, m.choose()
	case pickedBack:
		m.screen = musicBrowse
		m.list.scroll = 0
		m.reload()
		return m, nil
	case pickedQuery:
		m.query = event.text
		m.reload()
		return m, nil
	case pickedPlay:
		m.toggle()
		return m, nil
	case pickedStep:
		m.step(event.delta)
		return m, nil
	case pickedSeek:
		if m.now != nil && m.player != nil {
			m.resume = event.frame
			m.player.Play(m.now.Path, event.frame)
		}
		return m, nil
	case chosen:
		if event.err != nil {
			m.note = event.err.Error()
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
		m.reload()
		return m, gui.Tick()
	case window.Drop:
		return m, m.ingest(event.Paths)
	case window.Key, window.Pointer, window.Scroll:
		return m.delegate(event)
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
}

func (m *musicModel) delegate(msg gui.Msg) (gui.Model, gui.Cmd) {
	m.prepare()
	if cmd := take(&m.head, msg); cmd != nil {
		return m, cmd
	}
	switch m.screen {
	case musicNow:
		if cmd := take(&m.stage, msg); cmd != nil {
			return m, cmd
		}
	case musicAlbum:
		if cmd := take(&m.list, msg); cmd != nil {
			return m, cmd
		}
	default:
		if cmd := take(&m.shelf, msg); cmd != nil {
			return m, cmd
		}
		if cmd := take(&m.list, msg); cmd != nil {
			return m, cmd
		}
	}
	return m, nil
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
