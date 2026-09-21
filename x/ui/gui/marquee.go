package gui

import (
	"image"
	"math"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
)

const (
	marqueeItemH = 64
	marqueeGap   = 12
	marqueePad   = 20
	marqueeSpeed = 80
)

var marqueeColors = []Color{
	{220, 70, 90, 200},
	{70, 180, 140, 200},
	{70, 140, 220, 200},
	{230, 180, 60, 200},
	{180, 90, 210, 200},
	{50, 200, 200, 200},
}

// Bar is one scrolling rounded rect. It is a view value: Marquee.View
// builds bars from offset and size; nothing in View writes the model.
type Bar struct {
	color  Color
	y      float32
	width  float32
	period float32
}

func (b Bar) View() Node {
	return &Stack{Children: []Node{
		b.rect(b.y),
		b.rect(b.y - b.period),
	}}
}

func (b Bar) rect(y float32) Node {
	col := b.color
	return &Positioned{
		Y: y,
		Child: &Box{
			Width:  b.width,
			Height: marqueeItemH,
			Fill:   &col,
			Radius: 18,
		},
	}
}

func wrapShift(y, period float32) float32 {
	if period <= 0 {
		return y
	}
	y = float32(math.Mod(float64(y), float64(period)))
	if y < 0 {
		y += period
	}
	return y
}

type marqueeItem struct {
	at, wrap     *Positioned
	box, wrapBox *Box
	fill         Color
}

// Marquee is a clipped stack of [Bar] children that scroll down and wrap.
// Offset is the only motion state; View derives each bar from it.
type Marquee struct {
	size     image.Point
	offset   float32
	lastTick time.Duration
	lastY    int
	dragging bool
	paused   bool
	root     *Box
	bg       Color
	items    []marqueeItem
	treeN    int
	treeW    float32
	treeSize image.Point
}

// NewMarquee returns a looping color column. The error is always nil.
func NewMarquee() (*Marquee, error) {
	return &Marquee{size: image.Pt(800, 600)}, nil
}

func marqueeStride() float32 {
	return float32(marqueeItemH + marqueeGap)
}

func marqueeRepeats(innerH float32) int {
	period := float32(len(marqueeColors)) * marqueeStride()
	if period <= 0 {
		return 1
	}
	return max(1, int(math.Ceil(float64(innerH/period))))
}

func marqueeSlots(windowH int) int {
	inner := float32(windowH) - 2*marqueePad
	if inner < 1 {
		inner = 600
	}
	return 1 + 2*marqueeRepeats(inner)*len(marqueeColors)
}

func (m *Marquee) Init() Cmd { return Tick() }

func (m *Marquee) Update(msg Msg) (Model, Cmd) {
	if m == nil {
		return m, nil
	}
	var cmd Cmd
	if t, ok := msg.(TickMsg); ok {
		if !m.dragging && !m.paused && t.Elapsed > m.lastTick {
			m.offset = wrapShift(m.offset+float32((t.Elapsed-m.lastTick).Seconds())*marqueeSpeed, m.period())
		}
		m.lastTick = t.Elapsed
		cmd = Every(t.Period)
	}
	switch p := msg.(type) {
	case window.Pointer:
		if p.Button == 1 && p.Pressed {
			m.dragging = true
			m.lastY = p.Pos.Y
		}
		if p.Button == 1 && !p.Pressed {
			m.dragging = false
		}
		if m.dragging && p.Buttons&window.ButtonLeft != 0 {
			m.offset = wrapShift(m.offset+float32(p.Pos.Y-m.lastY), m.period())
			m.lastY = p.Pos.Y
		}
	case window.Scroll:
		m.offset = wrapShift(m.offset+float32(p.Delta.Y), m.period())
	case window.Key:
		if p.Pressed && !p.Repeat && (p.Rune == ' ' || p.Code == 49) {
			m.paused = !m.paused
		}
	case window.Resize:
		m.dragging = false
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		m.size = size
	}
	return m, cmd
}

func (m *Marquee) barCount() int {
	innerH := float32(m.size.Y) - 2*marqueePad
	if innerH < 1 {
		innerH = 1
	}
	n := marqueeRepeats(innerH) * len(marqueeColors)
	maxBars := (marqueeSlots(1200) - 1) / 2
	if maxBars < 1 {
		maxBars = 1
	}
	return max(1, min(n, maxBars))
}

func (m *Marquee) period() float32 {
	return float32(m.barCount()) * marqueeStride()
}

func (m *Marquee) barWidth() float32 {
	return max(float32(m.size.X)-2*marqueePad, 32)
}

func (m *Marquee) rebuild(n int, width float32) {
	items := make([]marqueeItem, n)
	children := make([]Node, n)
	for i := range items {
		col := marqueeColors[i%len(marqueeColors)]
		it := marqueeItem{
			fill:    col,
			at:      &Positioned{},
			wrap:    &Positioned{},
			box:     &Box{Width: width, Height: marqueeItemH, Radius: 18},
			wrapBox: &Box{Width: width, Height: marqueeItemH, Radius: 18},
		}
		it.box.Fill = &it.fill
		it.wrapBox.Fill = &it.fill
		it.at.Child = it.box
		it.wrap.Child = it.wrapBox
		children[i] = &Stack{Children: []Node{it.at, it.wrap}}
		items[i] = it
	}
	m.bg = Color{18, 18, 24, 255}
	m.root = &Box{
		Padding: EdgeInsets{marqueePad, marqueePad, marqueePad, marqueePad},
		Fill:    &m.bg,
		Clip:    true,
		Child:   &Stack{Clip: true, Children: children},
	}
	m.items = items
	m.treeN = n
	m.treeW = width
	m.treeSize = m.size
}

func (m *Marquee) View() Node {
	if m == nil {
		return nil
	}
	n := m.barCount()
	width := m.barWidth()
	if m.root == nil || m.treeN != n || m.treeW != width || m.treeSize != m.size {
		m.rebuild(n, width)
	}
	period := float32(n) * marqueeStride()
	for i := range m.items {
		y := wrapShift(m.offset+float32(i)*marqueeStride(), period)
		m.items[i].at.Y = y
		m.items[i].wrap.Y = y - period
	}
	return m.root
}
