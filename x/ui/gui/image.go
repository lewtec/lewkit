package gui

import (
	"image"
	"image/draw"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
	xdraw "golang.org/x/image/draw"
)

// Image paints Src scaled into the node. Radius punches a rounded mask.
// Width or Height 0 uses the source size on that axis.
type Image struct {
	Src    image.Image
	Width  float32
	Height float32
	Radius float32
	size   Size
}

type imageStamp struct {
	src    image.Image
	box    Rect
	clip   Rect
	radius float32
}

func (node *Image) Layout(constraints BoxConstraints) Size {
	if node == nil {
		return Size{}
	}
	width, height := node.Width, node.Height
	if node.Src != nil {
		bounds := node.Src.Bounds()
		if width == 0 {
			width = float32(bounds.Dx())
		}
		if height == 0 {
			height = float32(bounds.Dy())
		}
	}
	node.size = constraints.Constrain(Size{Width: width, Height: height})
	return node.size
}

func (node *Image) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if node == nil || picture == nil || node.Src == nil {
		return accumulatorOf(picture)
	}
	full := Rect{origin.X, origin.Y, node.size.Width, node.size.Height}
	visible := full.Intersect(clip)
	if visible.Width < 1 || visible.Height < 1 {
		return accumulatorOf(picture)
	}
	picture.blit(imageStamp{src: node.Src, box: full, clip: visible, radius: node.Radius})
	return accumulatorOf(picture)
}

func (stamp imageStamp) draw(destination *image.RGBA) {
	if destination == nil || stamp.src == nil || stamp.box.Width < 1 || stamp.box.Height < 1 {
		return
	}
	bounds := image.Rect(int(stamp.box.X), int(stamp.box.Y), int(stamp.box.X+stamp.box.Width), int(stamp.box.Y+stamp.box.Height))
	if bounds.Empty() || bounds.Intersect(destination.Bounds()).Empty() {
		return
	}
	scaled := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	xdraw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), stamp.src, stamp.src.Bounds(), draw.Src, nil)
	draw.Draw(destination, bounds, scaled, image.Point{}, draw.Over)
	clearOutside(destination, bounds, image.Rect(int(stamp.clip.X), int(stamp.clip.Y), int(stamp.clip.X+stamp.clip.Width), int(stamp.clip.Y+stamp.clip.Height)))
	if stamp.radius > 0 {
		punchRadius(destination, bounds, float64(stamp.radius))
	}
}

func clearOutside(destination *image.RGBA, bounds, keep image.Rectangle) {
	keep = keep.Intersect(bounds)
	pix := destination.Pix
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if y < destination.Bounds().Min.Y || y >= destination.Bounds().Max.Y {
			continue
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if x < destination.Bounds().Min.X || x >= destination.Bounds().Max.X {
				continue
			}
			if x >= keep.Min.X && x < keep.Max.X && y >= keep.Min.Y && y < keep.Max.Y {
				continue
			}
			off := destination.PixOffset(x, y)
			pix[off], pix[off+1], pix[off+2], pix[off+3] = 0, 0, 0, 0
		}
	}
}

// punchRadius clears pixels outside the rounded rect. Ink treats a zero
// alpha with leftover color as opaque, so the whole pixel goes to zero.
func punchRadius(destination *image.RGBA, bounds image.Rectangle, radius float64) {
	width := float64(bounds.Dx())
	height := float64(bounds.Dy())
	if width < 1 || height < 1 {
		return
	}
	if radius > width/2 {
		radius = width / 2
	}
	if radius > height/2 {
		radius = height / 2
	}
	pix := destination.Pix
	limit := destination.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if y < limit.Min.Y || y >= limit.Max.Y {
			continue
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if x < limit.Min.X || x >= limit.Max.X {
				continue
			}
			localX := float64(x) + 0.5 - float64(bounds.Min.X) - width/2
			localY := float64(y) + 0.5 - float64(bounds.Min.Y) - height/2
			halfW := width / 2
			halfH := height / 2
			cx := math.Abs(localX) - halfW + radius
			cy := math.Abs(localY) - halfH + radius
			ox := math.Max(cx, 0)
			oy := math.Max(cy, 0)
			corner := math.Max(cx, cy)
			inside := 0.0
			if corner < 0 {
				inside = corner
			}
			dist := inside + math.Hypot(ox, oy) - radius
			if dist > 0.5 {
				off := destination.PixOffset(x, y)
				pix[off] = 0
				pix[off+1] = 0
				pix[off+2] = 0
				pix[off+3] = 0
			}
		}
	}
}
