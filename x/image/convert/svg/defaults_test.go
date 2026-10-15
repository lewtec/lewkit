package svg

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewZerosAreSVGDefaults(t *testing.T) {
	t.Parallel()

	c := NewCircle()
	require.Equal(t, 0.0, c.CX())
	require.Equal(t, 0.0, c.CY())
	require.Equal(t, 0.0, c.R())
	col, ok := c.Fill()
	require.True(t, ok)
	assert.Equal(t, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, col)
	assert.Equal(t, 1.0, c.FillOpacity())
	assert.Equal(t, FillNonZero, c.FillRule())
	_, on := c.Stroke()
	assert.False(t, on)

	s := NewStroke()
	assert.Equal(t, color.NRGBA{A: 255}, s.Color())
	assert.Equal(t, 1.0, s.Opacity())
	assert.Equal(t, 1.0, s.Width())
	assert.Equal(t, CapButt, s.Cap())
	assert.Equal(t, JoinMiter, s.Join())
	assert.Equal(t, 4.0, s.MiterLimit())

	e := NewEllipse()
	require.Equal(t, 0.0, e.CX())
	require.Equal(t, 0.0, e.CY())
	require.Equal(t, 0.0, e.RX())
	require.Equal(t, 0.0, e.RY())
	r := NewRect()
	require.Equal(t, 0.0, r.X())
	require.Equal(t, 0.0, r.Y())
	require.Equal(t, 0.0, r.Width())
	require.Equal(t, 0.0, r.Height())
	require.Equal(t, 0.0, r.RX())
	require.Equal(t, 0.0, r.RY())
	d := NewDocument(256, 128)
	assert.Equal(t, 256.0, d.Width())
	assert.Equal(t, 128.0, d.Height())
	assert.False(t, d.ViewBox().Set())
}

func TestPublicConstructorsSetKind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  Kind
		want Kind
	}{
		{"group", NewGroup().Node().Kind(), KindGroup},
		{"circle", NewCircle().Node().Kind(), KindCircle},
		{"ellipse", NewEllipse().Node().Kind(), KindEllipse},
		{"rect", NewRect().Node().Kind(), KindRect},
		{"polygon", NewPolygon().Node().Kind(), KindPolygon},
		{"path", NewPath().Node().Kind(), KindPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.got)
			assert.NotEqual(t, KindInvalid, tt.got)
		})
	}
	assert.Equal(t, KindInvalid, (Node{}).Kind())
}

func TestWithFillAndOpacity(t *testing.T) {
	t.Parallel()
	c := NewCircle().WithFill(color.NRGBA{R: 255, A: 128})
	col, ok := c.Fill()
	require.True(t, ok)
	assert.Equal(t, uint8(255), col.R)
	assert.Equal(t, uint8(255), col.A)
	assert.Equal(t, 128.0/255, c.FillOpacity())
	c = c.WithFillOpacity(0.5)
	wantOp := float64(op8FromUnit(0.5)) / 255
	assert.Equal(t, wantOp, c.FillOpacity())
	c = c.WithFillNone()
	_, ok = c.Fill()
	assert.False(t, ok)
}

func TestStrokeColorSetsOpacity(t *testing.T) {
	t.Parallel()
	s := NewStroke().WithColor(color.NRGBA{B: 255, A: 64})
	assert.Equal(t, color.NRGBA{B: 255, A: 255}, s.Color())
	assert.Equal(t, 64.0/255, s.Opacity())
	s = s.WithOpacity(1)
	assert.Equal(t, 1.0, s.Opacity())
	c := NewCircle().WithStroke(s)
	got, on := c.Stroke()
	require.True(t, on)
	assert.Equal(t, 1.0, got.Width())
	c = c.WithoutStroke()
	_, on = c.Stroke()
	assert.False(t, on)
}

func TestFillOpacityClamp(t *testing.T) {
	t.Parallel()
	c := NewCircle().WithFillOpacity(-1)
	assert.Equal(t, 0.0, c.FillOpacity())
	c = NewCircle().WithFillOpacity(2)
	assert.Equal(t, 1.0, c.FillOpacity())
}
