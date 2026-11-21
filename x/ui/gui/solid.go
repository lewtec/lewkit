package gui

// Solid fills the frame with one RGBA color.
type Solid struct {
	fill Color
}

// NewSolid returns a solid fill. The error is always nil.
func NewSolid(red, green, blue, alpha uint8) (*Solid, error) {
	return &Solid{fill: Color{red, green, blue, alpha}}, nil
}

func (solid *Solid) Init() Cmd { return Tick() }

func (solid *Solid) Update(msg Msg) (Model, Cmd) {
	if solid == nil {
		return solid, nil
	}
	return solid, nil
}

func (solid *Solid) View() Node {
	if solid == nil {
		return nil
	}
	return &Box{Fill: &solid.fill}
}
