package gui

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

// Flex is Row/Column. Tight constraints fill the parent; loose constraints
// pack to children. Leftover main goes to Flex>0.
type Flex struct {
	Axis     Axis
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
	out := make([]FlexChild, len(children))
	for i, node := range children {
		out[i].Child = node
	}
	return &Flex{Axis: axis, Children: out}
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
	remaining := max(float32(0), maximumMain-usedMain)
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
	for i := range flex.Children {
		child := &flex.Children[i]
		child.position = cursor
		if child.Child != nil {
			cursor += axis.main(child.size)
		}
	}
	return flex.size
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

func (flex *Flex) Paint(origin Offset, clip Rect, paint *painter) {
	if flex == nil {
		return
	}
	for i := range flex.Children {
		child := &flex.Children[i]
		if child.Child == nil {
			continue
		}
		crossPadding := (flex.Axis.cross(flex.size) - flex.Axis.cross(child.size)) / 2
		if crossPadding < 0 {
			crossPadding = 0
		}
		paintChild(child.Child, origin.Add(flex.Axis.offset(child.position, crossPadding)), clip, paint)
	}
}
