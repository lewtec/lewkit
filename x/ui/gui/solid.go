package gui

// Solid fills the frame with one RGBA color.
type Solid struct {
	fill Color
}

// NewSolid returns a solid fill. The error is always nil.
func NewSolid(r, g, b, a uint8) (*Solid, error) {
	return &Solid{fill: Color{r, g, b, a}}, nil
}

func (s *Solid) Init() Cmd { return Tick() }

func (s *Solid) Update(msg Msg) (Model, Cmd) {
	if s == nil {
		return s, nil
	}
	return s, nil
}

func (s *Solid) View() Node {
	if s == nil {
		return nil
	}
	return &Box{Fill: &s.fill}
}
