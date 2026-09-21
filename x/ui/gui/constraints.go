package gui

const unbounded = float32(1e6)

// Size is a width and height in pixels.
type Size struct {
	Width, Height float32
}

// Offset is a paint translation.
type Offset struct {
	X, Y float32
}

func (offset Offset) Add(other Offset) Offset {
	return Offset{offset.X + other.X, offset.Y + other.Y}
}

// Rect is a clip or draw rectangle.
type Rect struct {
	X, Y, Width, Height float32
}

func (rectangle Rect) Intersect(other Rect) Rect {
	left := max(rectangle.X, other.X)
	top := max(rectangle.Y, other.Y)
	right := min(rectangle.X+rectangle.Width, other.X+other.Width)
	bottom := min(rectangle.Y+rectangle.Height, other.Y+other.Height)
	return Rect{left, top, max(float32(0), right-left), max(float32(0), bottom-top)}
}

// BoxConstraints is the Flutter box protocol: min/max width and height.
type BoxConstraints struct {
	MinWidth, MinHeight, MaxWidth, MaxHeight float32
}

// Tight is min = max = size.
func Tight(width, height float32) BoxConstraints {
	return BoxConstraints{width, height, width, height}
}

func (constraints BoxConstraints) Constrain(size Size) Size {
	return Size{
		clampFloat(size.Width, constraints.MinWidth, constraints.MaxWidth),
		clampFloat(size.Height, constraints.MinHeight, constraints.MaxHeight),
	}
}

func (constraints BoxConstraints) Deflate(padding EdgeInsets) BoxConstraints {
	return BoxConstraints{
		MinWidth:  max(float32(0), constraints.MinWidth-padding.Horizontal()),
		MinHeight: max(float32(0), constraints.MinHeight-padding.Vertical()),
		MaxWidth:  max(float32(0), constraints.MaxWidth-padding.Horizontal()),
		MaxHeight: max(float32(0), constraints.MaxHeight-padding.Vertical()),
	}
}

func (constraints BoxConstraints) Loosen() BoxConstraints {
	return BoxConstraints{0, 0, constraints.MaxWidth, constraints.MaxHeight}
}

func (constraints BoxConstraints) TightenWidth(width float32) BoxConstraints {
	width = clampFloat(width, constraints.MinWidth, constraints.MaxWidth)
	constraints.MinWidth, constraints.MaxWidth = width, width
	return constraints
}

func (constraints BoxConstraints) TightenHeight(height float32) BoxConstraints {
	height = clampFloat(height, constraints.MinHeight, constraints.MaxHeight)
	constraints.MinHeight, constraints.MaxHeight = height, height
	return constraints
}

func (constraints BoxConstraints) WithCross(axis Axis, minimum, maximum float32) BoxConstraints {
	if axis == Horizontal {
		constraints.MinHeight, constraints.MaxHeight = minimum, maximum
		return constraints
	}
	constraints.MinWidth, constraints.MaxWidth = minimum, maximum
	return constraints
}

func (constraints BoxConstraints) MainMax(axis Axis) float32 {
	if axis == Horizontal {
		return constraints.MaxWidth
	}
	return constraints.MaxHeight
}

func (constraints BoxConstraints) CrossMax(axis Axis) float32 {
	if axis == Horizontal {
		return constraints.MaxHeight
	}
	return constraints.MaxWidth
}

func (constraints BoxConstraints) tightMain(axis Axis) bool {
	if axis == Horizontal {
		return constraints.MinWidth == constraints.MaxWidth
	}
	return constraints.MinHeight == constraints.MaxHeight
}

func (constraints BoxConstraints) tightCross(axis Axis) bool {
	if axis == Horizontal {
		return constraints.MinHeight == constraints.MaxHeight
	}
	return constraints.MinWidth == constraints.MaxWidth
}

func clampFloat(value, low, high float32) float32 {
	return min(max(value, low), high)
}
