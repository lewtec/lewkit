package music

import (
	"image"

	"github.com/lewtec/lewkit/x/driver/window"
	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/lewtec/lewkit/x/ui/gui"
	"golang.org/x/image/font"
	"strconv"
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

func (s musicStyle) stack(children ...gui.Node) *gui.Flex {
	return gui.WithGap(s.px(8), gui.WithCross(gui.CrossStart, gui.Column(children...)))
}

func (s musicStyle) round(key, label string, size float32, fill, ink *gui.Color) *gui.Box {
	return &gui.Box{
		Key: key, Width: size, Height: size, Radius: size / 2, Fill: fill,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: s.line(label, ink),
	}
}

type headerModel struct {
	gui.Dirty
	musicStyle
	title    string
	showBack bool
	search   bool
	query    string
}

func (h *headerModel) Init() gui.Cmd { return nil }

func (h *headerModel) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if h == nil {
		return h, nil
	}
	if event, ok := msg.(window.Key); ok && event.Pressed && h.search {
		return h, h.key(event)
	}
	return h, nil
}

func (h *headerModel) key(key window.Key) gui.Cmd {
	switch {
	case key.Rune == 8 || key.Rune == 127 || key.Code == 22:
		if h.query != "" {
			runes := []rune(h.query)
			h.query = string(runes[:len(runes)-1])
			h.Dirty = gui.Touch(h.Dirty)
			return queryCmd(h.query)
		}
	case key.Rune == '\n' || key.Rune == '\r':
		h.Dirty, h.search = gui.See(h.Dirty, h.search, false)
	case key.Rune >= 32:
		h.query += string(key.Rune)
		h.Dirty = gui.Touch(h.Dirty)
		return queryCmd(h.query)
	}
	return nil
}

func queryCmd(text string) gui.Cmd {
	return func() gui.Msg { return pickedQuery{text: text} }
}

func (h *headerModel) View() gui.Node {
	if h == nil {
		return nil
	}
	lead := gui.Node(&gui.Box{Width: h.px(8)})
	if h.showBack {
		lead = h.round("back", "<", h.px(36), &h.paint.panel, &h.paint.text)
	}
	side := h.px(36)
	open := &gui.Box{
		Key: "open", Width: h.px(78), Height: side, Radius: h.px(12), Fill: &h.paint.panel,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: h.line("Open", &h.paint.text),
	}
	search := &gui.Box{
		Key: "search", Width: side, Height: side, Radius: side / 2, Fill: &h.paint.panel,
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: h.line("?", &h.paint.text),
	}
	heading := gui.Node(&gui.Box{})
	if h.title != "" {
		heading = h.titleText()
	}
	row := &gui.Flex{Axis: gui.Horizontal, Gap: h.px(8), Children: []gui.FlexChild{
		{Child: lead},
		gui.Expanded(&gui.Box{Height: h.px(48), Align: gui.Alignment{Y: 0.5}, Child: heading}),
		{Child: open},
		{Child: search},
	}}
	if h.search || h.query != "" {
		text := h.query
		if h.search {
			text += "_"
		}
		return h.stack(row, h.line(text, &h.paint.text))
	}
	return row
}

func (h *headerModel) titleText() gui.Node { return h.musicStyle.title(h.title) }

type albumModel struct {
	gui.Dirty
	musicStyle
	albums []Album
	inner  float32
	scroll float32
}

func (a *albumModel) Init() gui.Cmd { return nil }

func (a *albumModel) Update(gui.Msg) (gui.Model, gui.Cmd) {
	return a, nil
}

func (a *albumModel) View() gui.Node {
	if a == nil || len(a.albums) == 0 {
		return nil
	}
	cards := make([]gui.Node, len(a.albums))
	for i, album := range a.albums {
		cards[i] = a.card(album)
	}
	row := gui.WithGap(a.px(8), gui.Row(cards...))
	return &gui.Box{
		Key: "albums", Width: a.inner, Clip: true,
		Child: &gui.Positioned{X: -a.scroll, Child: row},
	}
}

func (a *albumModel) card(album Album) gui.Node {
	artSide := a.px(72)
	return &gui.Box{
		Key: "album:" + album.Name, Width: a.px(96), Radius: a.px(12), Fill: &a.paint.card,
		Padding: gui.EdgeInsets{Left: a.px(8), Top: a.px(8), Right: a.px(8), Bottom: a.px(8)},
		Child:   a.stack(a.thumb(artSide, a.px(10), album.Cover), a.muted(trim(album.Name, 12))),
	}
}

type trackModel struct {
	gui.Dirty
	musicStyle
	tracks   []Track
	inner    float32
	scroll   float32
	viewport float32
	busy     bool
}

func (t *trackModel) Init() gui.Cmd { return nil }

func (t *trackModel) Update(gui.Msg) (gui.Model, gui.Cmd) {
	return t, nil
}

func (t *trackModel) maxScroll() float32 {
	height := float32(len(t.tracks)) * (t.px(72) + t.px(8))
	return max(height-t.viewport, 0)
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
	rows := make([]gui.Node, len(t.tracks))
	for i, track := range t.tracks {
		rows[i] = t.row(track)
	}
	view := t.viewport
	if view < t.px(120) {
		view = t.px(120)
	}
	return &gui.Box{
		Key: "tracks", Width: t.inner, Height: view, Clip: true,
		Child: &gui.Positioned{Y: -t.scroll, Child: t.stack(rows...)},
	}
}

func (s musicStyle) thumb(side, radius float32, path string) gui.Node {
	if s.cover != nil {
		if img := s.cover(path); img != nil {
			return &gui.Image{Src: img, Width: side, Height: side, Radius: radius}
		}
	}
	return &gui.Box{Width: side, Height: side, Radius: radius, Fill: &s.paint.panel}
}

func (t *trackModel) row(track Track) gui.Node {
	artSide := t.px(48)
	return &gui.Box{
		Key: "track:" + strconv.FormatInt(track.ID, 10), Width: t.inner, Radius: t.px(12), Fill: &t.paint.card,
		Padding: gui.EdgeInsets{Left: t.px(10), Top: t.px(10), Right: t.px(10), Bottom: t.px(10)},
		Child: &gui.Flex{Axis: gui.Horizontal, Gap: t.px(12), Children: []gui.FlexChild{
			{Child: t.thumb(artSide, t.px(8), track.Cover)},
			gui.Expanded(&gui.Box{Height: artSide, Align: gui.Alignment{Y: 0.5}, Child: t.stack(
				t.line(track.Title, &t.paint.text),
				t.muted(track.Artist),
			)}),
		}},
	}
}

type nowModel struct {
	gui.Dirty
	musicStyle
	track   *Track
	inner   float32
	height  float32
	played  int64
	total   int64
	playing bool
}

func (n *nowModel) Init() gui.Cmd { return nil }

func (n *nowModel) Update(gui.Msg) (gui.Model, gui.Cmd) {
	return n, nil
}

func (n *nowModel) View() gui.Node {
	if n == nil {
		return nil
	}
	title, artist := "Nothing playing", ""
	side := n.squareSide()
	pad := n.px(20)
	content := side - pad*2
	if content < n.px(120) {
		content = n.px(120)
	}
	artSide := content - n.px(188)
	if artSide < n.px(96) {
		artSide = n.px(96)
	}
	if artSide > content {
		artSide = content
	}
	art := n.thumb(artSide, n.px(18), "")
	if n.track != nil {
		title = n.track.Title
		artist = n.track.Artist
		art = n.thumb(artSide, n.px(18), n.track.Cover)
	}
	frac := float32(0)
	if n.total > 0 {
		frac = float32(n.played) / float32(n.total)
	}
	if frac > 1 {
		frac = 1
	}
	barH := n.px(6)
	bar := &gui.Box{Key: "bar", Width: content, Height: n.px(28), Align: gui.Alignment{Y: 0.5}, Child: &gui.Stack{Children: []gui.Node{
		&gui.Box{Width: content, Height: barH, Radius: barH / 2, Fill: &n.paint.bar},
		&gui.Box{Width: max(frac*content, barH), Height: barH, Radius: barH / 2, Fill: &n.paint.text},
	}}}
	label := ">"
	if n.playing {
		label = "||"
	}
	play := n.round("play", label, n.px(64), &n.paint.text, &n.paint.onPlay)
	prev := n.round("prev", "|<", n.px(44), &n.paint.panel, &n.paint.text)
	next := n.round("next", ">|", n.px(44), &n.paint.panel, &n.paint.text)
	gap := n.px(18)
	left := clockDuration(framesToDuration(n.played, n.total, n.track))
	right := "0:00"
	if n.track != nil {
		right = clockDuration(n.track.Duration)
	}
	times := &gui.Flex{Axis: gui.Horizontal, Children: []gui.FlexChild{
		{Child: n.muted(left)},
		gui.Expanded(&gui.Box{}),
		{Child: n.muted(right)},
	}}
	transport := gui.WithGap(gap, gui.Row(prev, play, next))
	column := gui.WithGap(n.px(12), gui.Column(
		art,
		n.title(title),
		n.muted(artist),
		bar,
		&gui.Box{Width: content, Child: times},
		transport,
	))
	square := &gui.Box{
		Width: side, Height: side,
		Padding: gui.EdgeInsets{Left: pad, Top: pad, Right: pad, Bottom: pad},
		Align:   gui.Alignment{X: 0.5, Y: 0.5},
		Child:   column,
	}
	return &gui.Box{
		Width: n.inner, Height: n.room(),
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: square,
	}
}

func (n *nowModel) room() float32 {
	room := n.height - n.px(18)*2 - n.px(56)
	if room < n.px(220) {
		return n.px(220)
	}
	return room
}

func (n *nowModel) squareSide() float32 {
	side := n.inner
	if room := n.room(); room < side {
		side = room
	}
	if side < n.px(220) {
		return n.px(220)
	}
	return side
}
