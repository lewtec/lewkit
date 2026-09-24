package experiments

import (
	"fmt"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/ui/gui"
)

const (
	musicBaseW = 900
	musicBaseH = 700
)

type musicPaint struct {
	bg, panel, card, muted, text, onPlay, bar gui.Color
}

func paintFor(mode daynight.Mode) musicPaint {
	if mode == daynight.Light {
		return musicPaint{
			bg:     gui.Color{246, 246, 244, 255},
			panel:  gui.Color{232, 232, 228, 255},
			card:   gui.Color{255, 255, 255, 255},
			muted:  gui.Color{110, 110, 116, 255},
			text:   gui.Color{28, 28, 34, 255},
			onPlay: gui.Color{246, 246, 244, 255},
			bar:    gui.Color{210, 210, 206, 255},
		}
	}
	return musicPaint{
		bg:     gui.Color{16, 18, 46, 255},
		panel:  gui.Color{28, 32, 72, 255},
		card:   gui.Color{36, 40, 86, 255},
		muted:  gui.Color{168, 174, 206, 255},
		text:   gui.Color{255, 255, 255, 255},
		onPlay: gui.Color{20, 22, 40, 255},
		bar:    gui.Color{70, 74, 110, 255},
	}
}

// scale is 1 at 900×700. It follows the shorter side, including a retina
// surface whose pixel size is already a multiple of the point size.
func (m *musicModel) scale() float32 {
	if m == nil {
		return 1
	}
	w, h := float32(m.size.X), float32(m.size.Y)
	if w < 1 || h < 1 {
		return 1
	}
	return max(min(w/musicBaseW, h/musicBaseH), 0.75)
}

func (m *musicModel) px(n float32) float32 { return n * m.scale() }

func (m *musicModel) View() gui.Node {
	if m == nil {
		return nil
	}
	m.prepare()
	pad := m.px(18)
	inner := float32(m.size.X) - pad*2
	if inner < m.px(160) {
		inner = m.px(160)
	}
	return &gui.Box{
		Fill: &m.paint.bg,
		Child: &gui.Box{
			Height:  float32(m.size.Y),
			Padding: gui.EdgeInsets{Left: pad, Top: pad, Right: pad, Bottom: pad},
			Clip:    true,
			Child:   m.body(inner),
		},
	}
}

func (m *musicModel) body(inner float32) gui.Node {
	switch m.screen {
	case musicNow:
		return m.stage.stack(m.head.View(), m.stage.View())
	case musicAlbum:
		return m.list.stack(m.head.View(), m.albumHead(inner), m.list.View())
	default:
		nodes := []gui.Node{m.head.View()}
		if m.note != "" {
			nodes = append(nodes, m.list.muted(m.note))
		}
		if len(m.albums) > 0 {
			nodes = append(nodes, m.shelf.View())
		}
		nodes = append(nodes, m.list.View())
		return m.list.stack(nodes...)
	}
}

func (m *musicModel) albumHead(inner float32) gui.Node {
	coverH := m.px(180)
	var cover gui.Node = &gui.Box{Width: inner, Height: coverH, Radius: m.px(16), Fill: &m.paint.card}
	subtitle := "Unknown"
	count := 0
	for _, album := range m.albums {
		if album.Name != m.album {
			continue
		}
		subtitle = album.Artist
		count = album.Tracks
		if img := m.cover(album.Cover); img != nil {
			cover = &gui.Image{Src: img, Width: inner, Height: coverH, Radius: m.px(16)}
		}
	}
	return m.list.stack(
		cover,
		m.list.title(m.album),
		m.list.muted(fmt.Sprintf("%s  ·  %d", subtitle, count)),
	)
}

func (m *musicModel) prepare() {
	m.paint = paintFor(m.mode)
	style := musicStyle{paint: m.paint, scale: m.scale(), cover: m.cover}
	m.head.musicStyle = style
	m.shelf.musicStyle = style
	m.list.musicStyle = style
	m.stage.musicStyle = style
	m.head.title = "Browse"
	m.head.showBack = false
	if m.screen == musicAlbum {
		m.head.title = m.album
		m.head.showBack = true
	}
	if m.screen == musicNow {
		m.head.title = ""
		m.head.showBack = true
	}
	m.shelf.albums = m.albums
	m.shelf.inner = float32(m.size.X) - m.px(36)
	m.list.tracks = m.tracks
	m.list.inner = m.shelf.inner
	m.list.busy = m.busy
	m.list.viewport = m.listHeight(m.screen == musicBrowse && len(m.albums) > 0)
	played, total := m.progress()
	m.stage.track = m.now
	m.stage.inner = m.shelf.inner
	m.stage.height = float32(m.size.Y)
	m.stage.played = played
	m.stage.total = total
	m.stage.playing = m.player != nil && m.now != nil && m.player.Playing() && m.player.Path() == m.now.Path
}

func (m *musicModel) listHeight(albums bool) float32 {
	used := m.px(36) + m.px(56)
	if m.note != "" || m.head.search || m.head.query != "" {
		used += m.px(32)
	}
	if albums {
		used += m.px(160)
	}
	if m.screen == musicAlbum {
		used += m.px(220)
	}
	view := float32(m.size.Y) - used
	if view < m.px(180) {
		view = m.px(180)
	}
	return view
}

func (m *musicModel) progress() (played, total int64) {
	if m.player == nil || m.now == nil {
		return 0, 1
	}
	total = m.player.Total()
	if total <= 0 {
		return 0, 1
	}
	if m.player.Playing() && m.player.Path() == m.now.Path {
		return m.player.Played(), total
	}
	frame := m.resume
	if frame > total {
		frame = total
	}
	return frame, total
}

func framesToDuration(played, total int64, track *Track) time.Duration {
	if track == nil || total <= 0 || played <= 0 {
		return 0
	}
	return time.Duration(float64(played) / float64(total) * float64(track.Duration))
}

func trim(text string, n int) string {
	runes := []rune(text)
	if len(runes) <= n {
		return text
	}
	return string(runes[:n])
}

func clockDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	seconds := int(d.Round(time.Second) / time.Second)
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}
