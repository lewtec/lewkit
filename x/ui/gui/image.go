package gui

import (
	"hash/maphash"
	"image"
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
	picture.Image(node.Src, full, visible, node.Radius)
	return accumulatorOf(picture)
}

type thumbKey struct {
	src    uint64
	width  int
	height int
	radius uint32
}

// thumbSlot pairs a scaled copy with a hash of the source bytes.
// An animation reuses one *image.RGBA; a later frame replaces this slot
// when that hash changes.
type thumbSlot struct {
	rgba *image.RGBA
	sig  uint64
}

// pixSeed stays inside this process. imageSig is compared only here.
var pixSeed = maphash.MakeSeed()

// imageSig hashes pixels of the standard library images that can change
// without a new pointer. Every other image returns 0 and stays keyed by pointer.
func imageSig(src image.Image) uint64 {
	pix := mutablePix(src)
	if len(pix) == 0 {
		return 0
	}
	return maphash.Bytes(pixSeed, pix)
}

func mutablePix(src image.Image) []byte {
	switch src := src.(type) {
	case *image.Alpha:
		if src == nil {
			return nil
		}
		return src.Pix
	case *image.Alpha16:
		if src == nil {
			return nil
		}
		return src.Pix
	case *image.Gray:
		if src == nil {
			return nil
		}
		return src.Pix
	case *image.Gray16:
		if src == nil {
			return nil
		}
		return src.Pix
	case *image.RGBA:
		if src == nil {
			return nil
		}
		return src.Pix
	case *image.RGBA64:
		if src == nil {
			return nil
		}
		return src.Pix
	case *image.NRGBA:
		if src == nil {
			return nil
		}
		return src.Pix
	case *image.NRGBA64:
		if src == nil {
			return nil
		}
		return src.Pix
	case *image.CMYK:
		if src == nil {
			return nil
		}
		return src.Pix
	case *image.Paletted:
		if src == nil {
			return nil
		}
		return src.Pix
	default:
		return nil
	}
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
	width := keep.Dx() * 4
	for y := keep.Min.Y; y < keep.Max.Y; y++ {
		src := scaled.PixOffset(keep.Min.X-bounds.Min.X, y-bounds.Min.Y)
		dst := destination.PixOffset(keep.Min.X, y)
		copy(destination.Pix[dst:dst+width], scaled.Pix[src:src+width])
	}
}

func (picture *Picture) thumb(stamp imageStamp, width, height int) *image.RGBA {
	key := thumbKey{src: pointerOf(stamp.src), width: width, height: height, radius: math.Float32bits(stamp.radius)}
	sig := imageSig(stamp.src)
	if picture != nil {
		if got := picture.thumbs[key]; got.rgba != nil && got.sig == sig {
			return got.rgba
		}
	}
	// Ink stores straight alpha. The ink shader multiplies rgb by a.
	straight := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.ApproxBiLinear.Scale(straight, straight.Bounds(), stamp.src, stamp.src.Bounds(), xdraw.Src, nil)
	scaled := image.NewRGBA(straight.Bounds())
	copy(scaled.Pix, straight.Pix)
	if stamp.radius > 0 {
		punchRadius(scaled, scaled.Bounds(), scaled.Bounds(), float64(stamp.radius))
	}
	if picture == nil {
		return scaled
	}
	if picture.thumbs == nil {
		picture.thumbs = map[thumbKey]thumbSlot{}
	} else if _, exists := picture.thumbs[key]; !exists && len(picture.thumbs) > 256 {
		picture.thumbs = map[thumbKey]thumbSlot{}
	}
	picture.thumbs[key] = thumbSlot{rgba: scaled, sig: sig}
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
