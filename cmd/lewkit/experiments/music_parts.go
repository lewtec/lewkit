package experiments

import (
	"image"

	"github.com/lewtec/lewkit/x/driver/window"
	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/lewtec/lewkit/x/ui/gui"
	"golang.org/x/image/font"
)

type pickedAlbum struct{ name string }
type pickedTrack struct{ track Track }
type pickedOpen struct{}
type pickedBack struct{}
type pickedQuery struct{ text string }
type pickedPlay struct{}
type pickedStep struct{ delta int }
type pickedSeek struct{ frame int64 }

type musicStyle struct {
	paint musicPaint
	scale float32
	cover func(string) image.Image
}

func lewimageFace(scale, mul float32) font.Face {
	if scale == 0 {
		scale = 1
	}
	return lewimage.FaceSize(float64(18 * scale * mul))
}

func (s musicStyle) px(n float32) float32 {
	if s.scale == 0 {
		return n
	}
	return n * s.scale
}

func (s musicStyle) line(text string, color *gui.Color) gui.Node {
	ink := gui.Color{}
	if color != nil {
		ink = *color
	}
	return &gui.Text{Value: text, Ink: ink, Face: lewimageFace(s.scale, 1)}
}

func (s musicStyle) title(text string) gui.Node {
	return &gui.Text{Value: text, Ink: s.paint.text, Face: lewimageFace(s.scale, 1.7)}
}

func (s musicStyle) muted(text string) gui.Node {
	return &gui.Text{Value: text, Ink: s.paint.muted, Face: lewimageFace(s.scale, 0.85)}
}

func (s musicStyle) lead(children ...gui.Node) *gui.Flex {
	column := gui.Column(children...)
	column.Cross = gui.CrossStart
	return column
}

func (s musicStyle) round(label string, size float32, fill, ink *gui.Color) *gui.Box {
	return &gui.Box{
		Width: size, Height: size, Radius: size / 2, Fill: fill,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: s.line(label, ink),
	}
}

type headerModel struct {
	musicStyle
	title     string
	showBack  bool
	search    bool
	query     string
	back      *gui.Box
	open      *gui.Box
	searchBox *gui.Box
}

func (h *headerModel) Init() gui.Cmd { return nil }

func (h *headerModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if h == nil {
		return h, nil
	}
	switch event := msg.(type) {
	case window.Key:
		if event.Pressed && h.search {
			return h, h.key(event)
		}
	case window.Pointer:
		if event.Button == 1 && event.Pressed {
			return h, h.pointer(event.Pos)
		}
	}
	return h, nil
}

func (h *headerModel) key(key window.Key) gui.Cmd {
	switch {
	case key.Rune == 8 || key.Rune == 127 || key.Code == 22:
		if h.query != "" {
			runes := []rune(h.query)
			h.query = string(runes[:len(runes)-1])
			return queryCmd(h.query)
		}
	case key.Rune == '\n' || key.Rune == '\r':
		h.search = false
	case key.Rune >= 32:
		h.query += string(key.Rune)
		return queryCmd(h.query)
	}
	return nil
}

func queryCmd(text string) gui.Cmd {
	return func() gui.Msg { return pickedQuery{text: text} }
}

func (h *headerModel) pointer(pos image.Point) gui.Cmd {
	if h.open != nil && h.open.Contains(pos) {
		return func() gui.Msg { return pickedOpen{} }
	}
	if h.searchBox != nil && h.searchBox.Contains(pos) {
		h.search = true
		return nil
	}
	h.search = false
	if h.back != nil && h.showBack && h.back.Contains(pos) {
		return func() gui.Msg { return pickedBack{} }
	}
	return nil
}

func (h *headerModel) View() gui.Node {
	if h == nil {
		return nil
	}
	h.back, h.open, h.searchBox = nil, nil, nil
	var lead gui.Node
	if h.showBack {
		h.back = h.round("<", h.px(36), &h.paint.panel, &h.paint.text)
		lead = h.back
	} else {
		lead = &gui.Box{Width: h.px(8)}
	}
	side := h.px(36)
	h.open = &gui.Box{
		Width: h.px(78), Height: side, Radius: h.px(12), Fill: &h.paint.panel,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: h.line("Open", &h.paint.text),
	}
	h.searchBox = &gui.Box{
		Width: side, Height: side, Radius: side / 2, Fill: &h.paint.panel,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: h.line("?", &h.paint.text),
	}
	var heading gui.Node = &gui.Box{}
	if h.title != "" {
		heading = h.titleText()
	}
	row := &gui.Flex{Axis: gui.Horizontal, Children: []gui.FlexChild{
		{Child: lead},
		{Child: &gui.Box{Width: h.px(10)}},
		gui.Expanded(&gui.Box{Height: h.px(48), Align: gui.Alignment{Y: 0.5}, Child: heading}),
		{Child: h.open},
		{Child: &gui.Box{Width: h.px(8)}},
		{Child: h.searchBox},
	}}
	if h.search || h.query != "" {
		text := h.query
		if h.search {
			text += "_"
		}
		return h.lead(row, h.line(text, &h.paint.text))
	}
	return row
}

func (h *headerModel) titleText() gui.Node { return h.musicStyle.title(h.title) }

type albumModel struct {
	musicStyle
	albums []Album
	inner  float32
	scroll float32
	hits   []musicHit
	frame  *gui.Box
}

func (a *albumModel) Init() gui.Cmd { return nil }

func (a *albumModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if a == nil {
		return a, nil
	}
	switch event := msg.(type) {
	case window.Scroll:
		if a.frame != nil && a.frame.Contains(event.Pos) {
			a.scroll += float32(event.Delta.Y + event.Delta.X)
			if a.scroll < 0 {
				a.scroll = 0
			}
		}
	case window.Pointer:
		if event.Button == 1 && event.Pressed {
			for _, hit := range a.hits {
				if hit.box != nil && hit.box.Contains(event.Pos) {
					name := hit.album
					return a, func() gui.Msg { return pickedAlbum{name: name} }
				}
			}
		}
	}
	return a, nil
}

func (a *albumModel) View() gui.Node {
	if a == nil || len(a.albums) == 0 {
		return nil
	}
	a.hits = nil
	var cards []gui.Node
	for i := range a.albums {
		album := a.albums[i]
		artSide := a.px(72)
		var art gui.Node = &gui.Box{Width: artSide, Height: artSide, Radius: a.px(10), Fill: &a.paint.panel}
		if a.cover != nil {
			if img := a.cover(album.Cover); img != nil {
				art = &gui.Image{Src: img, Width: artSide, Height: artSide, Radius: a.px(10)}
			}
		}
		box := &gui.Box{
			Width: a.px(96), Radius: a.px(12), Fill: &a.paint.card,
			Padding: gui.EdgeInsets{Left: a.px(8), Top: a.px(8), Right: a.px(8), Bottom: a.px(8)},
			Child:   a.lead(art, &gui.Box{Height: a.px(4)}, a.muted(trim(album.Name, 12))),
		}
		a.hits = append(a.hits, musicHit{box: box, album: album.Name})
		cards = append(cards, box, &gui.Box{Width: a.px(8)})
	}
	a.frame = &gui.Box{
		Width: a.inner, Clip: true,
		Child: &gui.Positioned{X: -a.scroll, Child: gui.Row(cards...)},
	}
	return a.frame
}

type trackModel struct {
	musicStyle
	tracks   []Track
	inner    float32
	scroll   float32
	viewport float32
	busy     bool
	hits     []musicHit
	frame    *gui.Box
}

func (t *trackModel) Init() gui.Cmd { return nil }

func (t *trackModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if t == nil {
		return t, nil
	}
	switch event := msg.(type) {
	case window.Scroll:
		if t.frame == nil || !t.frame.Contains(event.Pos) {
			break
		}
		t.scroll += float32(event.Delta.Y)
		if t.scroll < 0 {
			t.scroll = 0
		}
		if t.scroll > t.maxScroll() {
			t.scroll = t.maxScroll()
		}
	case window.Pointer:
		if event.Button != 1 || !event.Pressed {
			break
		}
		for _, hit := range t.hits {
			if hit.box != nil && hit.box.Contains(event.Pos) && hit.track != nil {
				track := *hit.track
				return t, func() gui.Msg { return pickedTrack{track: track} }
			}
		}
	}
	return t, nil
}

func (t *trackModel) maxScroll() float32 {
	height := float32(len(t.hits)) * (t.px(72) + t.px(4))
	limit := height - t.viewport
	if limit < 0 {
		return 0
	}
	return limit
}

func (t *trackModel) View() gui.Node {
	if t == nil {
		return nil
	}
	if len(t.tracks) == 0 {
		if t.busy {
			return t.muted("reading library")
		}
		return t.muted("Drop a music folder")
	}
	t.hits = nil
	var rows []gui.Node
	for i := range t.tracks {
		track := t.tracks[i]
		artSide := t.px(48)
		var art gui.Node = &gui.Box{Width: artSide, Height: artSide, Radius: t.px(8), Fill: &t.paint.panel}
		if t.cover != nil {
			if img := t.cover(track.Cover); img != nil {
				art = &gui.Image{Src: img, Width: artSide, Height: artSide, Radius: t.px(8)}
			}
		}
		box := &gui.Box{
			Width: t.inner, Radius: t.px(12), Fill: &t.paint.card,
			Padding: gui.EdgeInsets{Left: t.px(10), Top: t.px(10), Right: t.px(10), Bottom: t.px(10)},
			Child: &gui.Flex{Axis: gui.Horizontal, Children: []gui.FlexChild{
				{Child: art},
				{Child: &gui.Box{Width: t.px(12)}},
				gui.Expanded(&gui.Box{Height: artSide, Align: gui.Alignment{Y: 0.5}, Child: t.lead(
					t.line(track.Title, &t.paint.text),
					t.muted(track.Artist),
				)}),
			}},
		}
		copy := track
		t.hits = append(t.hits, musicHit{box: box, track: &copy})
		rows = append(rows, box)
	}
	if t.scroll > t.maxScroll() {
		t.scroll = t.maxScroll()
	}
	view := t.viewport
	if view < t.px(120) {
		view = t.px(120)
	}
	t.frame = &gui.Box{
		Width: t.inner, Height: view, Clip: true,
		Child: &gui.Positioned{Y: -t.scroll, Child: t.lead(rows...)},
	}
	return t.frame
}

type nowModel struct {
	musicStyle
	track   *Track
	inner   float32
	height  float32
	played  int64
	total   int64
	playing bool
	play    *gui.Box
	prev    *gui.Box
	next    *gui.Box
	bar     *gui.Box
}

func (n *nowModel) Init() gui.Cmd { return nil }

func (n *nowModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if n == nil {
		return n, nil
	}
	event, ok := msg.(window.Pointer)
	if !ok || event.Button != 1 || !event.Pressed {
		return n, nil
	}
	switch {
	case n.play != nil && n.play.Contains(event.Pos):
		return n, func() gui.Msg { return pickedPlay{} }
	case n.prev != nil && n.prev.Contains(event.Pos):
		return n, func() gui.Msg { return pickedStep{delta: -1} }
	case n.next != nil && n.next.Contains(event.Pos):
		return n, func() gui.Msg { return pickedStep{delta: 1} }
	case n.bar != nil && n.bar.Contains(event.Pos) && n.total > 0:
		along, _ := n.bar.Unit(event.Pos)
		if along < 0 {
			along = 0
		}
		if along > 1 {
			along = 1
		}
		frame := int64(along * float32(n.total))
		return n, func() gui.Msg { return pickedSeek{frame: frame} }
	}
	return n, nil
}

func (n *nowModel) View() gui.Node {
	if n == nil {
		return nil
	}
	title, artist := "Nothing playing", ""
	artSide := n.px(280)
	if limit := n.height * 0.42; artSide > limit && limit > 0 {
		artSide = limit
	}
	if artSide > n.inner {
		artSide = n.inner
	}
	var art gui.Node = &gui.Box{Width: artSide, Height: artSide, Radius: n.px(18), Fill: &n.paint.card}
	if n.track != nil {
		title = n.track.Title
		artist = n.track.Artist
		if n.cover != nil {
			if img := n.cover(n.track.Cover); img != nil {
				art = &gui.Image{Src: img, Width: artSide, Height: artSide, Radius: n.px(18)}
			}
		}
	}
	frac := float32(0)
	if n.total > 0 {
		frac = float32(n.played) / float32(n.total)
	}
	if frac > 1 {
		frac = 1
	}
	barH := n.px(6)
	n.bar = &gui.Box{Width: n.inner, Height: n.px(28), Align: gui.Alignment{Y: 0.5}, Child: &gui.Stack{Children: []gui.Node{
		&gui.Box{Width: n.inner, Height: barH, Radius: barH / 2, Fill: &n.paint.bar},
		&gui.Box{Width: max(frac*n.inner, barH), Height: barH, Radius: barH / 2, Fill: &n.paint.text},
	}}}
	label := ">"
	if n.playing {
		label = "||"
	}
	n.play = n.round(label, n.px(64), &n.paint.text, &n.paint.onPlay)
	n.prev = n.round("|<", n.px(44), &n.paint.panel, &n.paint.text)
	n.next = n.round(">|", n.px(44), &n.paint.panel, &n.paint.text)
	gap := n.px(18)
	left := clockDuration(framesToDuration(n.played, n.total, n.track))
	right := "0:00"
	if n.track != nil {
		right = clockDuration(n.track.Duration)
	}
	return n.lead(
		&gui.Box{Height: gap},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: art},
		&gui.Box{Height: n.px(16)},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: n.title(title)},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: n.muted(artist)},
		&gui.Box{Height: n.px(16)},
		n.bar,
		&gui.Flex{Axis: gui.Horizontal, Children: []gui.FlexChild{
			{Child: n.muted(left)},
			gui.Expanded(&gui.Box{}),
			{Child: n.muted(right)},
		}},
		&gui.Box{Height: n.px(12)},
		&gui.Box{Align: gui.Alignment{X: 0.5}, Child: gui.Row(n.prev, &gui.Box{Width: gap}, n.play, &gui.Box{Width: gap}, n.next)},
	)
}
