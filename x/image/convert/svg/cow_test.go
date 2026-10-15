package svg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendCopyOnWrite(t *testing.T) {
	t.Parallel()
	d := NewDocument(10, 10)
	d2 := d.Append(NewCircle().Node())
	d3 := d.Append(NewRect().Node())
	assert.Empty(t, d.Children())
	if assert.Len(t, d2.Children(), 1) {
		assert.Equal(t, KindCircle, d2.Children()[0].Kind())
	}
	if assert.Len(t, d3.Children(), 1) {
		assert.Equal(t, KindRect, d3.Children()[0].Kind())
	}

	kids := d2.Children()
	kids[0] = NewEllipse().Node()
	assert.Equal(t, KindCircle, d2.Children()[0].Kind())

	g := NewGroup()
	g2 := g.Append(NewCircle().Node())
	g3 := g.Append(NewRect().Node())
	assert.Empty(t, g.Children())
	assert.Equal(t, KindCircle, g2.Children()[0].Kind())
	assert.Equal(t, KindRect, g3.Children()[0].Kind())
	gk := g2.Children()
	gk[0] = NewEllipse().Node()
	assert.Equal(t, KindCircle, g2.Children()[0].Kind())
}

func TestWithPointsCopies(t *testing.T) {
	t.Parallel()
	in := [][2]float64{{0, 0}, {10, 0}, {0, 10}}
	p, err := NewPolygon().WithPoints(in)
	require.NoError(t, err)
	in[0] = [2]float64{99, 99}
	got := p.Points()
	assert.Equal(t, [2]float64{0, 0}, got[0])
	got[1] = [2]float64{7, 7}
	again := p.Points()
	assert.Equal(t, [2]float64{10, 0}, again[1])
}

func TestWithPointsRejectsBadLength(t *testing.T) {
	t.Parallel()
	_, err := NewPolygon().WithPoints(nil)
	assert.ErrorIs(t, err, ErrPolygonPoints)
	two := [][2]float64{{0, 0}, {1, 1}}
	p, err := NewPolygon().WithPoints(two)
	assert.ErrorIs(t, err, ErrPolygonPoints)
	assert.Empty(t, p.Points())
	big := make([][2]float64, 1025)
	_, err = NewPolygon().WithPoints(big)
	assert.ErrorIs(t, err, ErrPolygonPoints)
	ok := make([][2]float64, 1024)
	_, err = NewPolygon().WithPoints(ok)
	assert.NoError(t, err)
}

func TestPathCommandsCopy(t *testing.T) {
	t.Parallel()
	in := []PathCmd{{Kind: CmdMove, X: 0, Y: 0}, {Kind: CmdLine, X: 1, Y: 0}, {Kind: CmdClose}}
	p, err := NewPath().WithCommands(in)
	require.NoError(t, err)
	in[0].X = 99
	got := p.Commands()
	assert.Equal(t, 0.0, got[0].X)
	got[1].X = 7
	again := p.Commands()
	assert.Equal(t, 1.0, again[1].X)
}

func TestRXIndependentOfRY(t *testing.T) {
	t.Parallel()
	r := NewRect().WithWidth(10).WithHeight(10).WithRX(5)
	assert.Equal(t, 5.0, r.RX())
	assert.Equal(t, 0.0, r.RY())
	rxSet, rySet := r.radiiSet()
	assert.True(t, rxSet)
	assert.False(t, rySet)
}
