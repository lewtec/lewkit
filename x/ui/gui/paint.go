package gui

import (
	"image"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
)

type slot struct {
	x, y, width, height     *ndarray.Tensor[float32]
	red, green, blue, alpha *ndarray.Tensor[float32]
	radius                  *ndarray.Tensor[float32]
	clipX, clipY            *ndarray.Tensor[float32]
	clipWidth, clipHeight   *ndarray.Tensor[float32]
}

// Picture is one over-composite tensor: each fill layers onto the
// accumulator, then ink. The graph grows with the tree; uniforms update
// each frame.
type Picture struct {
	slots       []slot
	composites  []*ndarray.Tensor[float32]
	base        *ndarray.Tensor[float32]
	accumulator *ndarray.Tensor[float32]
	pixelX      *ndarray.Tensor[float32]
	pixelY      *ndarray.Tensor[float32]
	channel     *ndarray.Tensor[int32]
	params      *ndarray.Tensor[float32]
	ink         *ndarray.Tensor[uint8]
	pixels      *ndarray.Tensor[uint8]
	inkedFrom   *ndarray.Tensor[float32]
	inkRGBA     *image.RGBA
	fills       []Draw
	texts       []textRun
	raster      *ndarray.Tensor[float32]
	black       *ndarray.Tensor[float32]
	signature   uint64
	hadInk      bool
	fillCount   int
	recordOnly  bool
}

func accumulatorOf(picture *Picture) *ndarray.Tensor[float32] {
	if picture == nil {
		return nil
	}
	return picture.accumulator
}

const slotFloats = 13

func (picture *Picture) cell(index int) (*ndarray.Tensor[float32], error) {
	tensor, err := picture.params.Shrink([][2]int{{index, index + 1}})
	if err != nil {
		return nil, err
	}
	return tensor.Splat()
}

func (picture *Picture) newSlot(base int) (slot, error) {
	var next slot
	fields := [slotFloats]**ndarray.Tensor[float32]{
		&next.x, &next.y, &next.width, &next.height, &next.red, &next.green, &next.blue, &next.alpha, &next.radius, &next.clipX, &next.clipY, &next.clipWidth, &next.clipHeight,
	}
	for offset, destination := range fields {
		tensor, err := picture.cell(base + offset)
		if err != nil {
			return next, err
		}
		*destination = tensor
	}
	return next, nil
}

func (slot slot) write(fill Draw) {
	writeUniform(slot.x, fill.X)
	writeUniform(slot.y, fill.Y)
	writeUniform(slot.width, fill.Width)
	writeUniform(slot.height, fill.Height)
	writeUniform(slot.red, fill.Red)
	writeUniform(slot.green, fill.Green)
	writeUniform(slot.blue, fill.Blue)
	writeUniform(slot.alpha, fill.Alpha)
	writeUniform(slot.radius, fill.Radius)
	writeUniform(slot.clipX, fill.ClipX)
	writeUniform(slot.clipY, fill.ClipY)
	writeUniform(slot.clipWidth, fill.ClipWidth)
	writeUniform(slot.clipHeight, fill.ClipHeight)
}

func writeUniform(tensor *ndarray.Tensor[float32], value float32) {
	if tensor == nil {
		return
	}
	buffer := tensor.Buffer()
	offset, ok, err := tensor.Tracker().At(0)
	if err != nil || !ok || offset < 0 || offset >= len(buffer) {
		return
	}
	buffer[offset] = value
}

// NewPicture starts from opaque black. Fills grow the over-composite as Paint returns.
func NewPicture() (*Picture, error) {
	picture := &Picture{}
	if err := picture.init(); err != nil {
		return nil, err
	}
	picture.pixels = picture.withInk(picture.base)
	picture.inkedFrom = picture.base
	return picture, nil
}

func (picture *Picture) init() error {
	ink, err := ndarray.New(make([]uint8, 4), ndarray.Shape{1, 1, 4})
	if err != nil {
		return err
	}
	shape := ndarray.Shape{1, 1, 4}
	picture.pixelX = ndarray.Coord(1, shape).Cast[float32]().Add(ndarray.Const(float32(0.5)))
	picture.pixelY = ndarray.Coord(0, shape).Cast[float32]().Add(ndarray.Const(float32(0.5)))
	picture.channel = ndarray.Coord(2, shape)
	picture.base = channelColor(picture.channel, ndarray.Const(float32(0)), ndarray.Const(float32(0)), ndarray.Const(float32(0)), ndarray.Const(float32(255)))
	picture.ink = ink
	picture.black = picture.base
	picture.accumulator = picture.base
	return nil
}

func (picture *Picture) withInk(accumulator *ndarray.Tensor[float32]) *ndarray.Tensor[uint8] {
	lit := picture.ink.Cast[int32]().CmpNe(ndarray.Const(int32(0)))
	return lit.Where(picture.ink.Cast[float32](), accumulator).Cast[uint8]()
}

func (picture *Picture) glyph(run textRun) {
	picture.texts = append(picture.texts, run)
}

func (picture *Picture) over(fill Draw) *ndarray.Tensor[float32] {
	if picture == nil {
		return nil
	}
	picture.fills = append(picture.fills, fill)
	if picture.recordOnly {
		picture.fillCount = len(picture.fills)
		return picture.base
	}
	index := len(picture.fills) - 1
	if index >= len(picture.slots) {
		capacity := len(picture.slots)
		if capacity < 1 {
			capacity = 1
		}
		for capacity <= index {
			capacity *= 2
		}
		if err := picture.compile(capacity); err != nil {
			return picture.accumulator
		}
		for previous := 0; previous < index; previous++ {
			picture.slots[previous].write(picture.fills[previous])
		}
	}
	picture.slots[index].write(fill)
	picture.fillCount = index + 1
	picture.accumulator = picture.composites[index]
	return picture.accumulator
}

func (picture *Picture) compile(count int) error {
	if picture == nil || count < 1 {
		return ndarray.ErrShape
	}
	params, err := ndarray.New(make([]float32, count*slotFloats), ndarray.Shape{count * slotFloats})
	if err != nil {
		return err
	}
	picture.params = params
	picture.slots = make([]slot, count)
	picture.composites = make([]*ndarray.Tensor[float32], count)
	accumulator := picture.base
	for index := 0; index < count; index++ {
		next, err := picture.newSlot(index * slotFloats)
		if err != nil {
			return err
		}
		picture.slots[index] = next
		coverage := next.coverage(picture.pixelX, picture.pixelY)
		alpha := coverage.Mul(next.alpha.Mul(ndarray.Const(float32(1.0 / 255))))
		color := channelColor(picture.channel, next.red, next.green, next.blue, ndarray.Const(float32(255)))
		accumulator = accumulator.Add(alpha.Mul(color.Add(accumulator.Neg())))
		picture.composites[index] = accumulator
	}
	picture.inkedFrom = nil
	return nil
}

func channelColor(channel *ndarray.Tensor[int32], red, green, blue, alpha *ndarray.Tensor[float32]) *ndarray.Tensor[float32] {
	return channel.Equal(ndarray.Const(int32(0))).Where(red,
		channel.Equal(ndarray.Const(int32(1))).Where(green,
			channel.Equal(ndarray.Const(int32(2))).Where(blue, alpha)))
}

func (slot slot) coverage(pixelX, pixelY *ndarray.Tensor[float32]) *ndarray.Tensor[float32] {
	half := ndarray.Const(float32(0.5))
	zero := ndarray.Const(float32(0))
	localX := pixelX.Add(slot.x.Neg()).Add(half).Add(slot.width.Mul(half).Neg())
	localY := pixelY.Add(slot.y.Neg()).Add(half).Add(slot.height.Mul(half).Neg())
	halfWidth := slot.width.Mul(half)
	halfHeight := slot.height.Mul(half)
	radius := slot.radius.CmpLt(halfWidth).Where(slot.radius, halfWidth)
	radius = radius.CmpLt(halfHeight).Where(radius, halfHeight)
	cornerX := localX.Max(localX.Neg()).Add(halfWidth.Neg()).Add(radius)
	cornerY := localY.Max(localY.Neg()).Add(halfHeight.Neg()).Add(radius)
	outsideX := cornerX.Max(zero)
	outsideY := cornerY.Max(zero)
	cornerMax := cornerX.Max(cornerY)
	inside := cornerMax.CmpLt(zero).Where(cornerMax, zero)
	distance := inside.Add(outsideX.Mul(outsideX).Add(outsideY.Mul(outsideY)).Sqrt()).Add(radius.Neg())
	cover := distance.CmpLt(half).Where(ndarray.Const(float32(1)), zero)
	insideX := pixelX.GreaterEqual(slot.clipX).And(pixelX.CmpLt(slot.clipX.Add(slot.clipWidth)))
	insideY := pixelY.GreaterEqual(slot.clipY).And(pixelY.CmpLt(slot.clipY.Add(slot.clipHeight)))
	clipEnabled := slot.clipWidth.GreaterEqual(half)
	clipMask := clipEnabled.Where(insideX.And(insideY), ndarray.Const(int32(1)))
	return clipMask.Where(cover, zero)
}

// Render layouts root, then Paint returns the over-composite tensor.
func (picture *Picture) Render(root Node, size Size) (*ndarray.Tensor[uint8], error) {
	if picture == nil || picture.base == nil {
		return nil, ErrView
	}
	if root == nil || size.Width < 1 || size.Height < 1 {
		return nil, ndarray.ErrShape
	}
	root.Layout(Tight(size.Width, size.Height))
	picture.fillCount = 0
	picture.fills = picture.fills[:0]
	picture.texts = picture.texts[:0]
	picture.raster = nil
	if picture.black != nil && picture.base != picture.black {
		picture.base = picture.black
		picture.slots = nil
		picture.composites = nil
		picture.inkedFrom = nil
	}
	picture.accumulator = picture.base
	accumulator := root.Paint(Offset{}, Rect{0, 0, size.Width, size.Height}, picture)
	if !picture.recordOnly {
		picture.fuseRaster()
		accumulator = picture.accumulator
	}
	if picture.recordOnly {
		height, width := int(size.Height), int(size.Width)
		if err := picture.ensureInk(height, width, len(picture.texts)); err != nil {
			return nil, err
		}
		for _, run := range picture.texts {
			run.stamp(picture.inkRGBA)
		}
		picture.stamp(len(picture.texts) > 0)
		return nil, nil
	}
	if accumulator == nil {
		accumulator = picture.base
	}
	if picture.pixels == nil || picture.inkedFrom != accumulator {
		if picture.pixels != nil {
			_ = picture.pixels.Close()
		}
		picture.pixels = picture.withInk(accumulator)
		picture.inkedFrom = accumulator
	}
	height, width := int(size.Height), int(size.Width)
	if err := picture.pixels.Resize(ndarray.Shape{height, width, 4}); err != nil {
		return nil, err
	}
	if err := picture.ensureInk(height, width, len(picture.texts)); err != nil {
		return nil, err
	}
	for _, run := range picture.texts {
		run.stamp(picture.inkRGBA)
	}
	picture.stamp(len(picture.texts) > 0)
	return picture.pixels, nil
}

func (picture *Picture) frameSig() uint64 {
	if picture == nil {
		return 0
	}
	return picture.signature
}

func (picture *Picture) stamp(ink bool) {
	if picture == nil {
		return
	}
	hash := uint64(14695981039346656037)
	mix := func(value uint64) {
		hash ^= value
		hash *= 1099511628211
	}
	if picture.recordOnly {
		for _, fill := range picture.fills {
			for _, value := range []float32{fill.X, fill.Y, fill.Width, fill.Height, fill.Red, fill.Green, fill.Blue, fill.Alpha, fill.Radius, fill.ClipX, fill.ClipY, fill.ClipWidth, fill.ClipHeight} {
				mix(uint64(math.Float32bits(value)))
			}
		}
		if picture.inkRGBA != nil {
			mix(uint64(picture.inkRGBA.Rect.Dx()))
			mix(uint64(picture.inkRGBA.Rect.Dy()))
		}
		if ink && picture.inkRGBA != nil {
			for _, pixel := range picture.inkRGBA.Pix {
				mix(uint64(pixel))
			}
		}
		picture.signature = hash
		return
	}
	if picture.params != nil {
		for _, uniform := range picture.params.Buffer() {
			mix(uint64(math.Float32bits(uniform)))
		}
	}
	if picture.pixels != nil {
		for _, dimension := range picture.pixels.Shape() {
			mix(uint64(dimension))
		}
	}
	if ink && picture.inkRGBA != nil {
		for _, pixel := range picture.inkRGBA.Pix {
			mix(uint64(pixel))
		}
	}
	picture.signature = hash
}

func (picture *Picture) ensureInk(height, width, texts int) error {
	if picture == nil || picture.ink == nil || height < 1 || width < 1 {
		return ndarray.ErrShape
	}
	need := height * width * 4
	if err := picture.ink.EnsureCells(need); err != nil {
		return err
	}
	pixels := picture.ink.Buffer()[:need]
	same := picture.inkRGBA != nil && picture.inkRGBA.Rect.Dx() == width && picture.inkRGBA.Rect.Dy() == height
	if texts > 0 || picture.hadInk || !same {
		clear(pixels)
	}
	picture.hadInk = texts > 0
	picture.inkRGBA = &image.RGBA{Pix: pixels, Stride: width * 4, Rect: image.Rect(0, 0, width, height)}
	return nil
}

// Ink is the CPU glyph overlay for this frame. Call after Render.
func (picture *Picture) Ink() *image.RGBA {
	if picture == nil {
		return nil
	}
	return picture.inkRGBA
}
