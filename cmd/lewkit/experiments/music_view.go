package experiments

import (
	"fmt"
	"time"

	"github.com/lewtec/lewkit/x/driver/daynight"
	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/lewtec/lewkit/x/ui/gui"
	"golang.org/x/image/font"
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

func (m *musicModel) face(mul float32) font.Face {
	return lewimage.FaceSize(float64(18 * m.scale() * mul))
}

func (m *musicModel) View() gui.Node {
	if m == nil {
		return nil
	}
	m.rows = nil
	m.albumsHit = nil
	m.back, m.open, m.searchBox, m.play, m.prev, m.next, m.bar, m.cards = nil, nil, nil, nil, nil, nil, nil, nil
	m.paint = paintFor(m.mode)
	pad := m.px(18)
	width := float32(m.size.X)
	inner := width - pad*2
	if inner < m.px(160) {
		inner = m.px(160)
	}
	return &gui.Box{
		Fill: &m.paint.bg,
		Child: &gui.Box{
			Height:  float32(m.size.Y),
			Padding: gui.EdgeInsets{Left: pad, Top: pad, Right: pad, Bottom: pad},
			Clip:    true,
			Child:   m.screenView(inner),
		},
	}
}

func (m *musicModel) screenView(inner float32) gui.Node {
	switch m.screen {
	case musicNow:
		return m.nowView(inner)
	case musicAlbum:
		return m.albumView(inner)
	default:
		return m.browseView(inner)
	}
}

func (m *musicModel) browseView(inner float32) gui.Node {
	header := m.titleRow("Browse", false)
	var body []gui.Node
	body = append(body, header)
	if line := m.queryLine(); line != nil {
		body = append(body, line)
	}
	if m.note != "" {
		body = append(body, m.muted(m.note))
	}
	if len(m.albums) > 0 {
		body = append(body, m.albumRow(inner))
	}
	body = append(body, m.trackList(inner))
	return m.lead(body...)
}

func (m *musicModel) albumView(inner float32) gui.Node {
	header := m.titleRow(m.album, true)
	coverH := m.px(180)
	var cover gui.Node = &gui.Box{Width: inner, Height: coverH, Radius: m.px(16), Fill: &m.paint.card}
	for _, album := range m.albums {
		if album.Name == m.album && m.cover(album.Cover) != nil {
			cover = &gui.Image{Src: m.cover(album.Cover), Width: inner, Height: coverH, Radius: m.px(16)}
			break
		}
	}
	subtitle := "Unknown"
	count := 0
	for _, album := range m.albums {
		if album.Name == m.album {
			subtitle = album.Artist
			count = album.Tracks
		}
	}
	return m.lead(
		header,
		&gui.Box{Height: m.px(12)},
		cover,
		&gui.Box{Height: m.px(10)},
		m.title(m.album),
		m.muted(fmt.Sprintf("%s  ·  %d", subtitle, count)),
		&gui.Box{Height: m.px(8)},
		m.trackList(inner),
	)
}

func (m *musicModel) nowView(inner float32) gui.Node {
	title, artist := "Nothing playing", ""
	artSide := m.px(280)
	if limit := float32(m.size.Y) * 0.42; artSide > limit {
		artSide = limit
	}
	if artSide > inner {
		artSide = inner
	}
	var art gui.Node = &gui.Box{Width: artSide, Height: artSide, Radius: m.px(18), Fill: &m.paint.card}
	if m.now != nil {
		title = m.now.Title
		artist = m.now.Artist
		if img := m.cover(m.now.Cover); img != nil {
			art = &gui.Image{Src: img, Width: artSide, Height: artSide, Radius: m.px(18)}
		}
	}
	played, total := m.progress()
	playedClock := clockDuration(framesToDuration(played, total, m.now))
	totalClock := "0:00"
	if m.now != nil {
		totalClock = clockDuration(m.now.Duration)
	}
	frac := float32(0)
	if total > 0 {
		frac = float32(played) / float32(total)
	}
	if frac > 1 {
		frac = 1
	}
	barH := m.px(6)
	m.bar = &gui.Box{Width: inner, Height: m.px(28), Align: gui.Alignment{Y: 0.5}, Child: &gui.Stack{Children: []gui.Node{
		&gui.Box{Width: inner, Height: barH, Radius: barH / 2, Fill: &m.paint.bar},
		&gui.Box{Width: max(frac*inner, barH), Height: barH, Radius: barH / 2, Fill: &m.paint.text},
	}}}
	label := ">"
	if m.player != nil && m.now != nil && m.player.Playing() && m.player.Path() == m.now.Path {
		label = "||"
	}
	m.play = m.roundButton(label, m.px(64), &m.paint.text, &m.paint.onPlay)
	m.prev = m.roundButton("|<", m.px(44), &m.paint.panel, &m.paint.text)
	m.next = m.roundButton(">|", m.px(44), &m.paint.panel, &m.paint.text)
	gap := m.px(18)
	return gui.Column(
		m.titleRow("", true),
		&gui.Box{Height: gap},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: art},
		&gui.Box{Height: m.px(16)},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: m.title(title)},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: m.muted(artist)},
		&gui.Box{Height: m.px(16)},
		m.bar,
		m.timeRow(playedClock, totalClock),
		&gui.Box{Height: m.px(12)},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: gui.Row(m.prev, &gui.Box{Width: gap}, m.play, &gui.Box{Width: gap}, m.next)},
	)
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

func (m *musicModel) timeRow(left, right string) gui.Node {
	return &gui.Flex{Axis: gui.Horizontal, Children: []gui.FlexChild{
		{Child: m.muted(left)},
		gui.Expanded(&gui.Box{}),
		{Child: m.muted(right)},
	}}
}

func (m *musicModel) titleRow(title string, back bool) gui.Node {
	var lead gui.Node
	if back {
		m.back = m.roundButton("<", m.px(36), &m.paint.panel, &m.paint.text)
		lead = m.back
	} else {
		lead = &gui.Box{Width: m.px(8)}
	}
	side := m.px(36)
	m.searchBox = &gui.Box{
		Width: side, Height: side, Radius: side / 2, Fill: &m.paint.panel,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: m.line(searchGlyph(m), &m.paint.text),
	}
	var heading gui.Node = &gui.Box{}
	if title != "" {
		heading = m.title(title)
	}
	return &gui.Flex{Axis: gui.Horizontal, Children: []gui.FlexChild{
		{Child: lead},
		{Child: &gui.Box{Width: m.px(10)}},
		gui.Expanded(&gui.Box{Height: m.px(48), Align: gui.Alignment{Y: 0.5}, Child: heading}),
		{Child: m.openButton()},
		{Child: &gui.Box{Width: m.px(8)}},
		{Child: m.searchBox},
	}}
}

func (m *musicModel) openButton() *gui.Box {
	m.open = &gui.Box{
		Width: m.px(78), Height: m.px(36), Radius: m.px(12), Fill: &m.paint.panel,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: m.line("Open", &m.paint.text),
	}
	return m.open
}

func searchGlyph(*musicModel) string { return "?" }

func (m *musicModel) queryLine() gui.Node {
	if m == nil || (!m.search && m.query == "") {
		return nil
	}
	text := m.query
	if m.search {
		text += "_"
	}
	return m.line(text, &m.paint.text)
}

func (m *musicModel) albumRow(inner float32) gui.Node {
	var cards []gui.Node
	m.albumsHit = nil
	for i := range m.albums {
		album := m.albums[i]
		artSide := m.px(72)
		var art gui.Node = &gui.Box{Width: artSide, Height: artSide, Radius: m.px(10), Fill: &m.paint.panel}
		if img := m.cover(album.Cover); img != nil {
			art = &gui.Image{Src: img, Width: artSide, Height: artSide, Radius: m.px(10)}
		}
		box := &gui.Box{
			Width: m.px(88), Height: m.px(108), Radius: m.px(12), Fill: &m.paint.card,
			Padding: gui.EdgeInsets{Left: m.px(8), Top: m.px(8), Right: m.px(8), Bottom: m.px(6)},
			Child:   m.lead(art, &gui.Box{Height: m.px(4)}, m.muted(trim(album.Name, 12))),
		}
		m.albumsHit = append(m.albumsHit, musicHit{box: box, album: album.Name})
		cards = append(cards, box, &gui.Box{Width: m.px(8)})
	}
	row := &gui.Positioned{X: -m.cardX, Child: gui.Row(cards...)}
	m.cards = &gui.Box{Width: inner, Height: m.px(116), Clip: true, Child: row}
	return m.cards
}

func (m *musicModel) trackList(inner float32) gui.Node {
	if len(m.tracks) == 0 {
		if m.busy {
			return m.muted("reading library")
		}
		return m.muted("Drop a music folder")
	}
	var rows []gui.Node
	m.rows = nil
	var height float32
	for i := range m.tracks {
		track := m.tracks[i]
		artSide := m.px(48)
		rowH := m.px(72)
		var art gui.Node = &gui.Box{Width: artSide, Height: artSide, Radius: m.px(8), Fill: &m.paint.panel}
		if img := m.cover(track.Cover); img != nil {
			art = &gui.Image{Src: img, Width: artSide, Height: artSide, Radius: m.px(8)}
		}
		box := &gui.Box{
			Width: inner, Height: rowH, Radius: m.px(12), Fill: &m.paint.card,
			Padding: gui.EdgeInsets{Left: m.px(10), Top: m.px(10), Right: m.px(10), Bottom: m.px(10)},
			Child: &gui.Flex{Axis: gui.Horizontal, Children: []gui.FlexChild{
				{Child: art},
				{Child: &gui.Box{Width: m.px(12)}},
				gui.Expanded(&gui.Box{Height: artSide, Align: gui.Alignment{Y: 0.5}, Child: m.lead(
					m.line(track.Title, &m.paint.text),
					m.muted(track.Artist),
				)}),
			}},
		}
		copy := track
		m.rows = append(m.rows, musicHit{box: box, track: &copy})
		rows = append(rows, box)
		height += rowH + m.px(4)
	}
	view := m.listHeight(len(m.albums) > 0)
	if view > float32(m.size.Y) {
		view = float32(m.size.Y)
	}
	m.maxScroll = height - view
	if m.maxScroll < 0 {
		m.maxScroll = 0
	}
	if m.scroll > m.maxScroll {
		m.scroll = m.maxScroll
	}
	return &gui.Box{
		Width: inner, Height: view, Clip: true,
		Child: &gui.Positioned{Y: -m.scroll, Child: m.lead(rows...)},
	}
}

func (m *musicModel) lead(children ...gui.Node) *gui.Flex {
	column := gui.Column(children...)
	column.Cross = gui.CrossStart
	return column
}

func (m *musicModel) listHeight(albums bool) float32 {
	used := m.px(36) + m.px(56)
	if m.note != "" || m.search || m.query != "" {
		used += m.px(32)
	}
	if albums {
		used += m.px(132)
	}
	view := float32(m.size.Y) - used
	if view < m.px(180) {
		view = m.px(180)
	}
	return view
}

func (m *musicModel) roundButton(label string, size float32, fill, ink *gui.Color) *gui.Box {
	return &gui.Box{
		Width: size, Height: size, Radius: size / 2, Fill: fill,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: m.line(label, ink),
	}
}

func (m *musicModel) line(text string, color *gui.Color) gui.Node {
	ink := gui.Color{}
	if color != nil {
		ink = *color
	}
	return &gui.Text{Value: text, Ink: ink, Face: m.face(1)}
}

func (m *musicModel) title(text string) gui.Node {
	return &gui.Text{Value: text, Ink: m.paint.text, Face: m.face(1.7)}
}

func (m *musicModel) muted(text string) gui.Node {
	return &gui.Text{Value: text, Ink: m.paint.muted, Face: m.face(0.85)}
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
