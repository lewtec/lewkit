package main

import (
	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/lewtec/lewkit/x/ui/gui"
	"golang.org/x/image/font"
)

// cardNodes is the credits overlay as layout nodes.
// The picture paints the fills and the text. Nothing here writes pixels.
func cardNodes(w, h float32) gui.Node {
	if w < 8 || h < 8 {
		return &gui.Stack{}
	}
	sx := func(v float32) float32 { return v * w / 498 }
	sy := func(v float32) float32 { return v * h / 280 }
	ink := gui.RGB{16, 16, 16, 255}
	word := lewimage.FaceSize(float64(max(8, int(sy(22)))))
	body := lewimage.FaceSize(float64(max(8, int(sy(9)))))
	tiny := lewimage.FaceSize(float64(max(8, int(sy(8)))))
	nodes := []gui.Node{
		at(sx(34), sy(18), text("Cyberpunk", word, ink)),
		tracked(sx(72), sy(40), sx(2), "2077", tiny, ink),
		at(sx(36), sy(228), text("Thank you,", body, ink)),
		at(sx(36), sy(242), text("Marcin Iwiński", tiny, ink)),
		at(sx(118), sy(242), text("&", tiny, ink)),
		at(sx(136), sy(242), text("Adam Badowski", tiny, ink)),
		at(sx(36), sy(252), text("CO-FOUNDER", tiny, ink)),
		at(sx(136), sy(252), text("HEAD OF STUDIO", tiny, ink)),
		at(sx(420), sy(236), text("CD PROJEKT RED", tiny, ink)),
	}
	nodes = append(nodes, mark(sx(396), sy(230), sy(18))...)
	nodes = append(nodes, glitch(w, h)...)
	return &gui.Stack{Children: nodes}
}

func at(x, y float32, child gui.Node) gui.Node {
	return &gui.Positioned{X: x, Y: y, Child: child}
}

func text(value string, face font.Face, ink gui.RGB) *gui.Text {
	return &gui.Text{Value: value, Face: face, Ink: ink, Cursor: -1}
}

func tracked(x, y, gap float32, value string, face font.Face, ink gui.RGB) gui.Node {
	children := make([]gui.Node, 0, len(value))
	cursor := x
	for _, r := range value {
		children = append(children, at(cursor, y, text(string(r), face, ink)))
		cursor += float32(lewimage.Advance(r, face)) + gap
	}
	return &gui.Stack{Children: children}
}

func mark(x, y, s float32) []gui.Node {
	if s < 8 {
		s = 8
	}
	red := swatch(196, 28, 36)
	return []gui.Node{
		at(x+s*0.28, y, &gui.Box{Width: s * 0.28, Height: s * 0.72, Fill: red}),
		at(x, y+s*0.18, &gui.Box{Width: s * 0.42, Height: s * 0.22, Fill: red}),
		at(x+s*0.42, y+s*0.42, &gui.Box{Width: s * 0.46, Height: s * 0.24, Fill: red}),
	}
}

func glitch(w, h float32) []gui.Node {
	black := swatch(6, 6, 6)
	hole := swatch(246, 254, 12)
	base := h * 266 / 280
	const n = 48
	step := w / n
	nodes := make([]gui.Node, 0, n+3)
	for i := 0; i < n; i++ {
		top := base
		switch i % 6 {
		case 1:
			top -= h / 70
		case 3:
			top += h / 100
		case 4:
			top -= h / 50
		}
		if top < 0 {
			top = 0
		}
		nodes = append(nodes, at(float32(i)*step, top, &gui.Box{
			Width: step + 1, Height: h - top, Fill: black,
		}))
	}
	for _, p := range [][2]float32{{w / 9, 2}, {w / 2, 1}, {w * 3 / 4, 3}} {
		nodes = append(nodes, at(p[0], base+p[1], &gui.Box{
			Width: max(2, w/50), Height: max(2, h/90), Fill: hole,
		}))
	}
	return nodes
}

func swatch(r, g, b uint8) *gui.RGB {
	c := gui.RGB{r, g, b, 255}
	return &c
}
