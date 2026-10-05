package main

import (
	"image"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/lewtec/lewkit/x/ui/world"
)

var inkBanner = gui.RGB{204, 204, 204, 255}

func (s *screen) track(pointer window.Pointer) {
	if s == nil || s.sim == nil {
		return
	}
	x, y, hit := pickSquare(s.size, pointer.Pos.Sub(s.origin))
	cur := cursor{}
	if hit {
		cur = cursor{x: x, y: y, on: true}
	}
	world.Put(s.sim.World, cur)
	if pointer.Button == 1 && pointer.Pressed {
		world.Put(s.sim.World, click{x: cur.x, y: cur.y, hit: cur.on, down: true})
	}
}

func (s *screen) view() gui.Node {
	text := "Next move: White"
	var frame image.Image
	if s != nil {
		frame = s.frame
		if s.sim != nil {
			if line, ok := world.Read[banner](s.sim.World); ok && line.text != "" {
				text = line.text
			}
		}
	}
	return &gui.Stack{Children: []gui.Node{
		&gui.Image{Src: frame},
		&gui.Positioned{X: 16, Y: 12, Child: &gui.Text{Value: text, Ink: inkBanner, Cursor: -1}},
	}}
}

// inset is the root of the chess view. Hint keeps the board and the
// caption in the fully shown box. A navbar or notch stays black.
type inset struct {
	screen *screen
	hint   *gui.Hint
}

func (n *inset) Layout(constraints gui.BoxConstraints) gui.Size {
	if n == nil || n.screen == nil {
		return gui.Size{}
	}
	if n.hint == nil {
		n.hint = &gui.Hint{}
	}
	n.hint.Child = n.screen.view()
	size := n.hint.Layout(constraints)
	if n.screen.fit(n.hint.Shown()) {
		n.hint.Child = n.screen.view()
		size = n.hint.Layout(constraints)
	}
	return size
}

func (n *inset) Paint(origin gui.Offset, clip gui.Rect, picture *gui.Picture) *ndarray.Tensor[float32] {
	if n == nil || n.hint == nil {
		return nil
	}
	return n.hint.Paint(origin, clip, picture)
}

// fit paints the scene at the safe box. The host zones are only known
// once Hint has laid out, so a change paints and the caller lays out again.
func (s *screen) fit(shown gui.Rect) bool {
	if s == nil {
		return false
	}
	width := int(shown.Width + 0.5)
	height := int(shown.Height + 0.5)
	if width < 16 || height < 16 {
		return false
	}
	origin := image.Pt(int(shown.X+0.5), int(shown.Y+0.5))
	size := image.Pt(width, height)
	if s.fitted && origin == s.origin && size == s.size {
		return false
	}
	s.fitted = true
	s.origin = origin
	s.size = size
	s.paint()
	return true
}

// paint draws the scene into a tensor the size of the window.
func (s *screen) paint() {
	if s == nil {
		return
	}
	w, h := s.size.X, s.size.Y
	if w < 2 {
		w = 2
	}
	if h < 2 {
		h = 2
	}
	if s.frame == nil || s.frame.Bounds().Dx() != w || s.frame.Bounds().Dy() != h {
		s.frame = image.NewRGBA(image.Rect(0, 0, w, h))
	}
	n := w * h
	if cap(s.depth) < n {
		s.depth = make([]float32, n)
		s.cover = make([]uint8, n)
	}
	s.depth = s.depth[:n]
	s.cover = s.cover[:n]
	paintChess(s.frame, s.sim, s.depth, s.cover, &s.boxes)
	if s.picture == nil || len(s.picture.Shape()) != 3 || s.picture.Shape()[0] != h || s.picture.Shape()[1] != w {
		if s.picture != nil {
			_ = s.picture.Close()
		}
		next, err := ndarray.New(make([]float32, n*4), ndarray.Shape{h, w, 4})
		if err != nil {
			s.picture = nil
			return
		}
		s.picture = next
	}
	copyFrame(s.picture.Buffer(), s.frame.Pix, s.frame.Stride, w, h)
	s.fullPaints++
	s.drawn = s.geometry()
	s.tones = readShades(s.sim.World)
	s.caption = s.captionText()
}

// recolor writes new square colors into the pixels those squares own.
// A hover then skips the mesh raster.
func (s *screen) recolor(ids []int) {
	if s == nil || s.frame == nil || s.picture == nil || len(s.cover) == 0 {
		s.paint()
		return
	}
	gain := squareLit()
	var col [64][3]byte
	for _, id := range ids {
		if id < 0 || id >= len(col) {
			continue
		}
		src := colDark
		switch s.tones[id] {
		case shadeHover:
			src = colHover
		case shadeSelected:
			src = colSelected
		case shadeLight:
			src = colLight
		}
		col[id] = [3]byte{sat(src.r * gain), sat(src.g * gain), sat(src.b * gain)}
	}
	w := s.frame.Bounds().Dx()
	stride := s.frame.Stride
	pix := s.frame.Pix
	buf := s.picture.Buffer()
	for _, id := range ids {
		if id < 0 || id >= len(s.boxes) {
			continue
		}
		box := s.boxes[id]
		if box.empty() {
			continue
		}
		mark := byte(id + 1)
		c := col[id]
		cr, cg, cb := float32(c[0]), float32(c[1]), float32(c[2])
		for y := box.minY; y <= box.maxY; y++ {
			row := pix[y*stride:]
			cov := s.cover[y*w:]
			dst := buf[y*w*4 : (y+1)*w*4]
			for x := box.minX; x <= box.maxX; x++ {
				if cov[x] != mark {
					continue
				}
				o := x * 4
				row[o], row[o+1], row[o+2] = c[0], c[1], c[2]
				dst[o], dst[o+1], dst[o+2] = cr, cg, cb
			}
		}
	}
}

func copyFrame(dst []float32, pix []byte, stride, w, h int) {
	for y := 0; y < h; y++ {
		row := pix[y*stride : y*stride+w*4]
		out := dst[y*w*4 : (y+1)*w*4]
		for i, p := range row {
			out[i] = float32(p)
		}
	}
}

func readShades(w *world.World) (out [64]shade) {
	if w == nil {
		return out
	}
	shades := world.Query[shade](w)
	world.Query[square](w).Read(func(e world.Entity, sq square) {
		if sq.x > 7 || sq.y > 7 {
			return
		}
		if got, ok := shades.Get(e); ok {
			out[int(sq.x)*8+int(sq.y)] = got
		}
	})
	return out
}
