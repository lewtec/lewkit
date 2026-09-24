package gui

import "github.com/lewtec/lewkit/x/ndarray"

// Axis is the main axis of a [Flex].
type Axis int

const (
	Horizontal Axis = iota
	Vertical
)

func (axis Axis) main(size Size) float32 {
	if axis == Horizontal {
		return size.Width
	}
	return size.Height
}

func (axis Axis) cross(size Size) float32 {
	if axis == Horizontal {
		return size.Height
	}
	return size.Width
}

func (axis Axis) size(main, cross float32) Size {
	if axis == Horizontal {
		return Size{Width: main, Height: cross}
	}
	return Size{Width: cross, Height: main}
}

func (axis Axis) offset(main, cross float32) Offset {
	if axis == Horizontal {
		return Offset{main, cross}
	}
	return Offset{cross, main}
}

// FlexChild is a [Flex] slot. Flex > 0 takes leftover main-axis space.
type FlexChild struct {
	Child Node
	Flex  float32

	size     Size
	position float32
}

// CrossAlign places children on the cross axis. The zero value centers.
type CrossAlign int

const (
	CrossCenter CrossAlign = iota
	CrossStart
	CrossEnd
)

// Flex is Row/Column. Tight constraints fill the parent; loose constraints
// pack to children. Leftover main goes to Flex>0. Gap is the space between children.
type Flex struct {
	Axis     Axis
	Cross    CrossAlign
	Gap      float32
	Children []FlexChild

	size Size
}

// Expanded is a flex child that takes leftover main-axis space.
func Expanded(child Node) FlexChild {
	return FlexChild{Child: child, Flex: 1}
}

// Row is a horizontal [Flex].
func Row(children ...Node) *Flex {
	return newFlex(Horizontal, children)
}

// Column is a vertical [Flex].
func Column(children ...Node) *Flex {
	return newFlex(Vertical, children)
}

func newFlex(axis Axis, children []Node) *Flex {
	items := make([]FlexChild, len(children))
	for i, node := range children {
		items[i].Child = node
	}
	return &Flex{Axis: axis, Children: items}
}

func (flex *Flex) Layout(constraints BoxConstraints) Size {
	if flex == nil {
		return Size{}
	}
	axis := flex.Axis
	maximumMain := constraints.MainMax(axis)
	maximumCross := constraints.CrossMax(axis)
	var usedMain, totalFlex, maxChildCross float32
	for i := range flex.Children {
		child := &flex.Children[i]
		if child.Child == nil || child.Flex > 0 {
			totalFlex += max(float32(0), child.Flex)
			continue
		}
		child.size = child.Child.Layout(axis.box(0, maximumMain-usedMain, 0, maximumCross))
		usedMain += axis.main(child.size)
		maxChildCross = max(maxChildCross, axis.cross(child.size))
	}
	remaining := max(float32(0), maximumMain-usedMain-flex.gaps())
	for i := range flex.Children {
		child := &flex.Children[i]
		if child.Child == nil || child.Flex <= 0 {
			continue
		}
		shareMain := remaining
		if totalFlex > 0 {
			shareMain = remaining * child.Flex / totalFlex
		}
		child.size = child.Child.Layout(axis.box(shareMain, shareMain, 0, maximumCross))
		usedMain += axis.main(child.size)
		maxChildCross = max(maxChildCross, axis.cross(child.size))
	}
	usedMain += flex.gaps()
	mainSize := usedMain
	if constraints.tightMain(axis) {
		mainSize = maximumMain
	}
	crossSize := maxChildCross
	if constraints.tightCross(axis) {
		crossSize = maximumCross
	}
	flex.size = constraints.Constrain(axis.size(max(mainSize, constraints.mainMinimum(axis)), crossSize))
	var cursor float32
	placed := 0
	for i := range flex.Children {
		child := &flex.Children[i]
		if child.Child == nil {
			continue
		}
		if placed > 0 {
			cursor += flex.Gap
		}
		child.position = cursor
		cursor += axis.main(child.size)
		placed++
	}
	return flex.size
}

func (flex *Flex) gaps() float32 {
	if flex == nil || flex.Gap == 0 {
		return 0
	}
	count := 0
	for i := range flex.Children {
		if flex.Children[i].Child != nil {
			count++
		}
	}
	if count < 2 {
		return 0
	}
	return flex.Gap * float32(count-1)
}

func (constraints BoxConstraints) mainMinimum(axis Axis) float32 {
	if axis == Horizontal {
		return constraints.MinWidth
	}
	return constraints.MinHeight
}

func (axis Axis) box(minMain, maxMain, minCross, maxCross float32) BoxConstraints {
	if axis == Horizontal {
		return BoxConstraints{minMain, minCross, maxMain, maxCross}
	}
	return BoxConstraints{minCross, minMain, maxCross, maxMain}
}

func (flex *Flex) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if flex == nil {
		return accumulatorOf(picture)
	}
	accumulator := accumulatorOf(picture)
	for i := range flex.Children {
		child := &flex.Children[i]
		if child.Child == nil {
			continue
		}
		space := flex.Axis.cross(flex.size) - flex.Axis.cross(child.size)
		if space < 0 {
			space = 0
		}
		var crossPadding float32
		switch flex.Cross {
		case CrossStart:
			crossPadding = 0
		case CrossEnd:
			crossPadding = space
		default:
			crossPadding = space / 2
		}
		accumulator = child.Child.Paint(origin.Add(flex.Axis.offset(child.position, crossPadding)), clip, picture)
	}
	return accumulator
}
