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

type thumbKey struct {
	src    uint64
	width  int
	height int
	radius uint32
}

func (stamp imageStamp) draw(picture *Picture, destination *image.RGBA) {
	if destination == nil || stamp.src == nil || stamp.box.Width < 1 || stamp.box.Height < 1 {
		return
	}
	bounds := image.Rect(int(stamp.box.X), int(stamp.box.Y), int(stamp.box.X+stamp.box.Width), int(stamp.box.Y+stamp.box.Height))
	if bounds.Empty() || bounds.Intersect(destination.Bounds()).Empty() {
		return
	}
	keep := bounds.Intersect(image.Rect(int(stamp.clip.X), int(stamp.clip.Y), int(stamp.clip.X+stamp.clip.Width), int(stamp.clip.Y+stamp.clip.Height)))
	keep = keep.Intersect(destination.Bounds())
	if keep.Empty() {
		return
	}
	scaled := picture.thumb(stamp, bounds.Dx(), bounds.Dy())
	draw.Draw(destination, keep, scaled, image.Pt(keep.Min.X-bounds.Min.X, keep.Min.Y-bounds.Min.Y), draw.Over)
}

func (picture *Picture) thumb(stamp imageStamp, width, height int) *image.RGBA {
	key := thumbKey{src: pointerOf(stamp.src), width: width, height: height, radius: math.Float32bits(stamp.radius)}
	if picture != nil {
		if got := picture.thumbs[key]; got != nil {
			return got
		}
	}
	scaled := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.ApproxBiLinear.Scale(scaled, scaled.Bounds(), stamp.src, stamp.src.Bounds(), draw.Src, nil)
	if stamp.radius > 0 {
		punchRadius(scaled, scaled.Bounds(), scaled.Bounds(), float64(stamp.radius))
	}
	if picture == nil {
		return scaled
	}
	if picture.thumbs == nil || len(picture.thumbs) > 256 {
		picture.thumbs = map[thumbKey]*image.RGBA{}
	}
	picture.thumbs[key] = scaled
	return scaled
}

// punchRadius clears pixels outside the rounded rect. Ink treats a zero
// alpha with leftover color as opaque, so the whole pixel goes to zero.
func punchRadius(destination *image.RGBA, bounds, keep image.Rectangle, radius float64) {
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
	for y := keep.Min.Y; y < keep.Max.Y; y++ {
		for x := keep.Min.X; x < keep.Max.X; x++ {
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
