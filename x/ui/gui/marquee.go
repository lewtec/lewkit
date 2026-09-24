package gui

import (
	"image"
	"math"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
)

const (
	marqueeItemHeight = 64
	marqueeGap        = 12
	marqueePad        = 20
	marqueeSpeed      = 80
	marqueeStride     = float32(marqueeItemHeight + marqueeGap)
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

func (bar Bar) View() Node {
	return &Stack{Children: []Node{
		bar.rect(bar.y),
		bar.rect(bar.y - bar.period),
	}}
}

func (bar Bar) rect(y float32) Node {
	color := bar.color
	return &Positioned{
		Y: y,
		Child: &Box{
			Width:  bar.width,
			Height: marqueeItemHeight,
			Fill:   &color,
			Radius: 18,
		},
	}
}

func (marquee *Marquee) wrap(y float32) float32 {
	return wrapShift(y, marquee.period())
}

func (marquee *Marquee) shift(delta float32) {
	next := marquee.wrap(marquee.offset + delta)
	Set(marquee, &marquee.offset, next)
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
	Dirty
	size         image.Point
	offset       float32
	lastTick     time.Duration
	lastPointerY int
	dragging     bool
	paused       bool
	root         *Box
	background   Color
	items        []marqueeItem
	treeCount    int
	treeWidth    float32
	treeSize     image.Point
}

// NewMarquee returns a looping color column. The error is always nil.
func NewMarquee() (*Marquee, error) {
	return &Marquee{size: image.Pt(800, 600)}, nil
}

func (*Marquee) repeats(innerHeight float32) int {
	period := float32(len(marqueeColors)) * marqueeStride
	if period <= 0 {
		return 1
	}
	return max(1, int(math.Ceil(float64(innerHeight/period))))
}

func (marquee *Marquee) Init() Cmd { return Tick() }

func (marquee *Marquee) Update(msg Msg) (Model, Cmd) {
	if marquee == nil {
		return marquee, nil
	}
	var cmd Cmd
	if tick, ok := msg.(TickMsg); ok {
		if !marquee.dragging && !marquee.paused && tick.Elapsed > marquee.lastTick {
			marquee.shift(float32((tick.Elapsed - marquee.lastTick).Seconds()) * marqueeSpeed)
		}
		marquee.lastTick = tick.Elapsed
		cmd = Every(tick.Period)
	}
	switch event := msg.(type) {
	case window.Pointer:
		if event.Button == 1 && event.Pressed {
			Set(marquee, &marquee.dragging, true)
			marquee.lastPointerY = event.Pos.Y
		}
		if event.Button == 1 && !event.Pressed {
			Set(marquee, &marquee.dragging, false)
		}
		if marquee.dragging && event.Buttons&window.ButtonLeft != 0 {
			marquee.shift(float32(event.Pos.Y - marquee.lastPointerY))
			marquee.lastPointerY = event.Pos.Y
		}
	case window.Scroll:
		marquee.shift(float32(event.Delta.Y))
	case window.Key:
		if event.Pressed && !event.Repeat && (event.Rune == ' ' || event.Code == 49) {
			Set(marquee, &marquee.paused, !marquee.paused)
		}
	case window.Resize:
		Set(marquee, &marquee.dragging, false)
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		Set(marquee, &marquee.size, size)
	}
	return marquee, cmd
}

func (marquee *Marquee) barCount() int {
	innerHeight := float32(marquee.size.Y) - 2*marqueePad
	if innerHeight < 1 {
		innerHeight = 1
	}
	return max(1, marquee.repeats(innerHeight)*len(marqueeColors))
}

func (marquee *Marquee) period() float32 {
	return float32(marquee.barCount()) * marqueeStride
}

func (marquee *Marquee) barWidth() float32 {
	return max(float32(marquee.size.X)-2*marqueePad, 32)
}

func (marquee *Marquee) rebuild(count int, width float32) {
	items := make([]marqueeItem, count)
	children := make([]Node, count)
	for i := range items {
		color := marqueeColors[i%len(marqueeColors)]
		item := marqueeItem{
			fill:    color,
			at:      &Positioned{},
			wrap:    &Positioned{},
			box:     &Box{Width: width, Height: marqueeItemHeight, Radius: 18},
			wrapBox: &Box{Width: width, Height: marqueeItemHeight, Radius: 18},
		}
		item.box.Fill = &item.fill
		item.wrapBox.Fill = &item.fill
		item.at.Child = item.box
		item.wrap.Child = item.wrapBox
		children[i] = &Stack{Children: []Node{item.at, item.wrap}}
		items[i] = item
	}
	marquee.background = Color{18, 18, 24, 255}
	marquee.root = &Box{
		Padding: EdgeInsets{marqueePad, marqueePad, marqueePad, marqueePad},
		Fill:    &marquee.background,
		Clip:    true,
		Child:   &Stack{Clip: true, Children: children},
	}
	marquee.items = items
	marquee.treeCount = count
	marquee.treeWidth = width
	marquee.treeSize = marquee.size
}

func (marquee *Marquee) View() Node {
	if marquee == nil {
		return nil
	}
	count := marquee.barCount()
	width := marquee.barWidth()
	if marquee.root == nil || marquee.treeCount != count || marquee.treeWidth != width || marquee.treeSize != marquee.size {
		marquee.rebuild(count, width)
	}
	period := marquee.period()
	for i := range marquee.items {
		y := marquee.wrap(marquee.offset + float32(i)*marqueeStride)
		marquee.items[i].at.Y = y
		marquee.items[i].wrap.Y = y - period
	}
	return marquee.root
}
