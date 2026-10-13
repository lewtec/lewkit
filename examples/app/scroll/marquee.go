package main

import (
	"image"
	"math"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ui/gui"
)

const (
	marqueeItemHeight = 64
	marqueeGap        = 12
	marqueePad        = 20
	marqueeSpeed      = 80
	marqueeStride     = float32(marqueeItemHeight + marqueeGap)
)

var marqueeColors = []gui.RGB{
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
	color  gui.RGB
	y      float32
	width  float32
	period float32
}

func (bar Bar) View() gui.Node {
	return &gui.Stack{Children: []gui.Node{
		bar.rect(bar.y),
		bar.rect(bar.y - bar.period),
	}}
}

func (bar Bar) rect(y float32) gui.Node {
	color := bar.color
	return &gui.Positioned{
		Y: y,
		Child: &gui.Box{
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
	marquee.Dirty, marquee.offset = gui.See(marquee.Dirty, marquee.offset, next)
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
	at, wrap     *gui.Positioned
	box, wrapBox *gui.Box
	fill         gui.RGB
}

// Marquee is a clipped stack of [Bar] children that scroll down and wrap.
// Offset is the only motion state; View derives each bar from it.
type Marquee struct {
	gui.Dirty
	size         image.Point
	offset       float32
	lastTick     time.Duration
	lastPointerY int
	dragging     bool
	paused       bool
	root         *gui.Box
	background   gui.RGB
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

func (marquee *Marquee) Init() gui.Cmd { return gui.Tick() }

func (marquee *Marquee) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if marquee == nil {
		return marquee, nil
	}
	var cmd gui.Cmd
	if tick, ok := msg.(gui.TickMsg); ok {
		if !marquee.dragging && !marquee.paused && tick.Elapsed > marquee.lastTick {
			marquee.shift(float32((tick.Elapsed - marquee.lastTick).Seconds()) * marqueeSpeed)
		}
		marquee.lastTick = tick.Elapsed
		cmd = gui.Every(tick.Period)
	}
	switch event := msg.(type) {
	case window.Pointer:
		if event.Button == 1 && event.Pressed {
			marquee.Dirty, marquee.dragging = gui.See(marquee.Dirty, marquee.dragging, true)
			marquee.lastPointerY = event.Pos.Y
		}
		if event.Button == 1 && !event.Pressed {
			marquee.Dirty, marquee.dragging = gui.See(marquee.Dirty, marquee.dragging, false)
		}
		if marquee.dragging && event.Buttons&window.ButtonLeft != 0 {
			marquee.shift(float32(event.Pos.Y - marquee.lastPointerY))
			marquee.lastPointerY = event.Pos.Y
		}
	case window.Scroll:
		marquee.shift(float32(event.Delta.Y))
	case window.Key:
		if event.Pressed && !event.Repeat && (event.Rune == ' ' || event.Code == 49) {
			marquee.Dirty, marquee.paused = gui.See(marquee.Dirty, marquee.paused, !marquee.paused)
		}
	case window.Resize:
		marquee.Dirty, marquee.dragging = gui.See(marquee.Dirty, marquee.dragging, false)
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		marquee.Dirty, marquee.size = gui.See(marquee.Dirty, marquee.size, size)
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
	children := make([]gui.Node, count)
	for i := range items {
		color := marqueeColors[i%len(marqueeColors)]
		item := marqueeItem{
			fill:    color,
			at:      &gui.Positioned{},
			wrap:    &gui.Positioned{},
			box:     &gui.Box{Width: width, Height: marqueeItemHeight, Radius: 18},
			wrapBox: &gui.Box{Width: width, Height: marqueeItemHeight, Radius: 18},
		}
		item.box.Fill = &item.fill
		item.wrapBox.Fill = &item.fill
		item.at.Child = item.box
		item.wrap.Child = item.wrapBox
		children[i] = &gui.Stack{Children: []gui.Node{item.at, item.wrap}}
		items[i] = item
	}
	marquee.background = gui.RGB{18, 18, 24, 255}
	marquee.root = &gui.Box{
		Padding: gui.EdgeInsets{marqueePad, marqueePad, marqueePad, marqueePad},
		Fill:    &marquee.background,
		Clip:    true,
		Child:   &gui.Stack{Clip: true, Children: children},
	}
	marquee.items = items
	marquee.treeCount = count
	marquee.treeWidth = width
	marquee.treeSize = marquee.size
}

func (marquee *Marquee) View() gui.Node {
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

func sizeOf(msg gui.Msg) (image.Point, bool) {
	switch message := msg.(type) {
	case gui.TickMsg:
		return message.Size, true
	case window.Resize:
		return message.Size, true
	default:
		return image.Point{}, false
	}
}
