package experiments

import (
	"context"
	"image"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
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
	lib, err := OpenLibrary()
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

type musicModel struct {
	ctx       context.Context
	lib       *Library
	player    *player
	size      image.Point
	dir       string
	query     string
	search    bool
	screen    musicScreen
	album     string
	albums    []Album
	tracks    []Track
	now       *Track
	resume    int64
	scroll    float32
	cardX     float32
	busy      bool
	booted    bool
	note      string
	maxScroll float32
	covers    map[string]image.Image

	back      *gui.Box
	searchBox *gui.Box
	play      *gui.Box
	prev      *gui.Box
	next      *gui.Box
	bar       *gui.Box
	cards     *gui.Box
	rows      []musicHit
	albumsHit []musicHit
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
	case window.Key:
		if event.Pressed && m.search {
			m.key(event)
		}
	case window.Pointer:
		if event.Button == 1 && event.Pressed {
			m.pointer(event.Pos)
		}
	case window.Scroll:
		if m.cards != nil && m.cards.Contains(event.Pos) {
			m.cardX += float32(event.Delta.Y + event.Delta.X)
			if m.cardX < 0 {
				m.cardX = 0
			}
			break
		}
		m.scroll += float32(event.Delta.Y)
		if m.scroll < 0 {
			m.scroll = 0
		}
		if m.maxScroll > 0 && m.scroll > m.maxScroll {
			m.scroll = m.maxScroll
		}
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

func (m *musicModel) key(key window.Key) {
	switch {
	case key.Rune == 8 || key.Rune == 127 || key.Code == 22:
		if m.query != "" {
			runes := []rune(m.query)
			m.query = string(runes[:len(runes)-1])
			m.reload()
		}
	case key.Rune == '\n' || key.Rune == '\r':
		m.search = false
	case key.Rune >= 32:
		m.query += string(key.Rune)
		m.reload()
	}
}

func (m *musicModel) pointer(pos image.Point) {
	if m.searchBox != nil && m.searchBox.Contains(pos) {
		m.search = true
		return
	}
	m.search = false
	if m.back != nil && m.back.Contains(pos) {
		m.screen = musicBrowse
		m.scroll = 0
		m.reload()
		return
	}
	if m.play != nil && m.play.Contains(pos) {
		m.toggle()
		return
	}
	if m.prev != nil && m.prev.Contains(pos) {
		m.step(-1)
		return
	}
	if m.next != nil && m.next.Contains(pos) {
		m.step(1)
		return
	}
	if m.bar != nil && m.bar.Contains(pos) && m.now != nil && m.player != nil && m.player.Total() > 0 {
		along, _ := m.bar.Unit(pos)
		if along < 0 {
			along = 0
		}
		if along > 1 {
			along = 1
		}
		frame := int64(along * float32(m.player.Total()))
		m.resume = frame
		m.player.Play(m.now.Path, frame)
		return
	}
	for _, hit := range m.albumsHit {
		if hit.box != nil && hit.box.Contains(pos) {
			m.album = hit.album
			m.screen = musicAlbum
			m.scroll = 0
			m.reload()
			return
		}
	}
	for _, hit := range m.rows {
		if hit.box != nil && hit.box.Contains(pos) && hit.track != nil {
			m.start(*hit.track)
			return
		}
	}
}

func (m *musicModel) start(track Track) {
	m.now = &track
	m.resume = 0
	m.screen = musicNow
	m.scroll = 0
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
