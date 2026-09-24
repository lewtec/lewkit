package experiments

import (
	"fmt"
	"time"

	"github.com/lewtec/lewkit/x/ui/gui"
)

var (
	musicBg    = gui.Color{16, 18, 46, 255}
	musicPanel = gui.Color{28, 32, 72, 255}
	musicCard  = gui.Color{36, 40, 86, 255}
	musicMuted = gui.Color{168, 174, 206, 255}
	musicWhite = gui.Color{255, 255, 255, 255}
	musicInk   = gui.Color{20, 22, 40, 255}
	musicBar   = gui.Color{70, 74, 110, 255}
)

func (m *musicModel) View() gui.Node {
	if m == nil {
		return nil
	}
	m.rows = nil
	m.albumsHit = nil
	m.back, m.searchBox, m.play, m.prev, m.next, m.bar, m.cards = nil, nil, nil, nil, nil, nil, nil
	width := float32(m.size.X)
	if width > 440 {
		width = 440
	}
	if width < 280 {
		width = float32(m.size.X)
	}
	return &gui.Box{
		Fill:  &musicBg,
		Align: gui.Alignment{X: 0.5},
		Child: &gui.Box{
			Width:   width,
			Height:  float32(m.size.Y),
			Padding: gui.EdgeInsets{Left: 18, Top: 16, Right: 18, Bottom: 16},
			Clip:    true,
			Child:   m.screenView(width - 36),
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
	return gui.Column(body...)
}

func (m *musicModel) albumView(inner float32) gui.Node {
	header := m.titleRow(m.album, true)
	var cover gui.Node = &gui.Box{Width: inner, Height: 180, Radius: 16, Fill: &musicCard}
	for _, album := range m.albums {
		if album.Name == m.album && m.cover(album.Cover) != nil {
			cover = &gui.Image{Src: m.cover(album.Cover), Width: inner, Height: 180, Radius: 16}
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
	return gui.Column(
		header,
		&gui.Box{Height: 12},
		cover,
		&gui.Box{Height: 10},
		m.line(m.album, &musicWhite),
		m.muted(fmt.Sprintf("%s  ·  %d", subtitle, count)),
		&gui.Box{Height: 8},
		m.trackList(inner),
	)
}

func (m *musicModel) nowView(inner float32) gui.Node {
	title, artist := "Nothing playing", ""
	var art gui.Node = &gui.Box{Width: 220, Height: 220, Radius: 18, Fill: &musicCard}
	if m.now != nil {
		title = m.now.Title
		artist = m.now.Artist
		if img := m.cover(m.now.Cover); img != nil {
			art = &gui.Image{Src: img, Width: 220, Height: 220, Radius: 18}
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
	m.bar = &gui.Box{Width: inner, Height: 28, Align: gui.Alignment{Y: 0.5}, Child: &gui.Stack{Children: []gui.Node{
		&gui.Box{Width: inner, Height: 4, Radius: 2, Fill: &musicBar},
		&gui.Box{Width: max(frac*inner, 4), Height: 4, Radius: 2, Fill: &musicWhite},
	}}}
	label := ">"
	if m.player != nil && m.now != nil && m.player.Playing() && m.player.Path() == m.now.Path {
		label = "||"
	}
	m.play = m.roundButton(label, 64, &musicWhite, &musicInk)
	m.prev = m.roundButton("|<", 44, &musicPanel, &musicWhite)
	m.next = m.roundButton(">|", 44, &musicPanel, &musicWhite)
	return gui.Column(
		m.titleRow("", true),
		&gui.Box{Height: 18},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: art},
		&gui.Box{Height: 16},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: m.line(title, &musicWhite)},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: m.muted(artist)},
		&gui.Box{Height: 16},
		m.bar,
		m.timeRow(playedClock, totalClock),
		&gui.Box{Height: 12},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: gui.Row(m.prev, &gui.Box{Width: 18}, m.play, &gui.Box{Width: 18}, m.next)},
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
		m.back = m.roundButton("<", 36, &musicPanel, &musicWhite)
		lead = m.back
	} else {
		lead = &gui.Box{Width: 8}
	}
	m.searchBox = &gui.Box{
		Width: 36, Height: 36, Radius: 18, Fill: &musicPanel,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: m.line(searchGlyph(m), &musicWhite),
	}
	var heading gui.Node = &gui.Box{}
	if title != "" {
		heading = m.line(title, &musicWhite)
	}
	return &gui.Flex{Axis: gui.Horizontal, Children: []gui.FlexChild{
		{Child: lead},
		{Child: &gui.Box{Width: 10}},
		gui.Expanded(&gui.Box{Height: 40, Align: gui.Alignment{Y: 0.5}, Child: heading}),
		{Child: m.searchBox},
	}}
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
	return m.line(text, &musicWhite)
}

func (m *musicModel) albumRow(inner float32) gui.Node {
	var cards []gui.Node
	m.albumsHit = nil
	for i := range m.albums {
		album := m.albums[i]
		var art gui.Node = &gui.Box{Width: 72, Height: 72, Radius: 10, Fill: &musicPanel}
		if img := m.cover(album.Cover); img != nil {
			art = &gui.Image{Src: img, Width: 72, Height: 72, Radius: 10}
		}
		box := &gui.Box{
			Width: 88, Height: 108, Radius: 12, Fill: &musicCard,
			Padding: gui.EdgeInsets{Left: 8, Top: 8, Right: 8, Bottom: 6},
			Child:   gui.Column(art, &gui.Box{Height: 4}, m.muted(trim(album.Name, 10))),
		}
		m.albumsHit = append(m.albumsHit, musicHit{box: box, album: album.Name})
		cards = append(cards, box, &gui.Box{Width: 8})
	}
	row := &gui.Positioned{X: -m.cardX, Child: gui.Row(cards...)}
	m.cards = &gui.Box{Width: inner, Height: 116, Clip: true, Child: row}
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
		var art gui.Node = &gui.Box{Width: 44, Height: 44, Radius: 8, Fill: &musicPanel}
		if img := m.cover(track.Cover); img != nil {
			art = &gui.Image{Src: img, Width: 44, Height: 44, Radius: 8}
		}
		box := &gui.Box{
			Width: inner, Height: 60, Radius: 12, Fill: &musicCard,
			Padding: gui.EdgeInsets{Left: 8, Top: 8, Right: 8, Bottom: 8},
			Child: gui.Row(
				art,
				&gui.Box{Width: 10},
				&gui.Box{Height: 44, Align: gui.Alignment{Y: 0.5}, Child: gui.Column(
					m.line(track.Title, &musicWhite),
					m.muted(track.Artist),
				)},
			),
		}
		copy := track
		m.rows = append(m.rows, musicHit{box: box, track: &copy})
		rows = append(rows, box)
		height += 64
	}
	view := float32(m.size.Y) - 220
	if view < 120 {
		view = 120
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
		Child: &gui.Positioned{Y: -m.scroll, Child: gui.Column(rows...)},
	}
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
	return &gui.Text{Value: text, Ink: ink}
}

func (m *musicModel) muted(text string) gui.Node {
	return &gui.Text{Value: text, Ink: musicMuted}
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
