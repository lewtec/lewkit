package svg

import (
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeGoldens(t *testing.T) {
	t.Parallel()
	poly, err := NewPolygon().WithPoints([][2]float64{{0, 0}, {10, 0}, {0, 10}})
	require.NoError(t, err)
	tests := []struct {
		name string
		doc  Document
	}{
		{name: "empty.svg", doc: NewDocument(256, 256)},
		{
			name: "rx-only.svg",
			doc: NewDocument(256, 256).Append(
				NewRect().WithWidth(10).WithHeight(10).WithRX(5).Node(),
			),
		},
		{
			name: "circle.svg",
			doc: NewDocument(256, 256).Append(
				NewCircle().WithCX(50).WithCY(50).WithR(40).Node(),
			),
		},
		{
			name: "paint.svg",
			doc: NewDocument(256, 256).Append(
				NewRect().WithWidth(100).WithHeight(80).
					WithFill(color.NRGBA{R: 255, A: 255}).
					WithFillOpacity(0.5).
					WithFillRule(FillEvenOdd).
					WithStroke(NewStroke().WithWidth(2).WithCap(CapRound)).
					Node(),
			),
		},
		{
			name: "nested.svg",
			doc: NewDocument(256, 256).Append(
				NewGroup().Append(NewCircle().WithR(10).Node()).Node(),
			),
		},
		{name: "polygon.svg", doc: NewDocument(256, 256).Append(poly.Node())},
		{
			name: "path-tri.svg",
			doc: NewDocument(256, 256).Append(
				NewPath().MoveTo(40, 40).LineTo(120, 40).LineTo(80, 100).Close().Node(),
			),
		},
		{
			name: "path-cubic.svg",
			doc: NewDocument(256, 256).Append(
				NewPath().MoveTo(40, 120).CubicTo(40, 40, 120, 40, 120, 120).CubicTo(120, 200, 40, 200, 40, 120).Close().Node(),
			),
		},
		{
			name: "path-stroke.svg",
			doc: NewDocument(256, 256).Append(
				NewPath().MoveTo(30, 180).LineTo(125, 40).LineTo(220, 180).
					WithFillNone().
					WithStroke(NewStroke().WithWidth(4)).
					Node(),
			),
		},
		{name: "viewbox.svg", doc: NewDocument(256, 256).WithViewBox(0, 0, 100, 50)},
		{
			name: "linear.svg",
			doc: NewDocument(64, 16).Append(
				NewRect().WithWidth(64).WithHeight(16).
					WithLinearFill(NewLinearFill(0, 8, 64, 8,
						color.NRGBA{R: 255, A: 255},
						color.NRGBA{A: 255},
					)).Node(),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := EncodeToString(tt.doc)
			require.NoError(t, err)
			want := readGolden(t, tt.name)
			assert.Equal(t, want, got)
		})
	}
}

func TestEncodeOmitsNegZero(t *testing.T) {
	t.Parallel()
	neg := math.Copysign(0, -1)
	doc := NewDocument(256, 256).Append(NewCircle().WithCX(neg).WithCY(neg).WithR(1).Node())
	got, err := EncodeToString(doc)
	require.NoError(t, err)
	assert.False(t, containsMinusZero(got))
}

func TestEncodeOpacityRoundTrip(t *testing.T) {
	t.Parallel()
	for u := range 256 {
		s := encodeOpacity(uint8(u))
		require.Equal(t, uint8(u), op8FromUnit(mustFloat(s)))
	}
}

func TestEncodeRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		doc  Document
	}{
		{name: "zero size", doc: NewDocument(0, 256)},
		{name: "non-integer", doc: NewDocument(256.5, 256)},
		{name: "too big", doc: NewDocument(8192, 256)},
		{name: "negative r", doc: NewDocument(256, 256).Append(NewCircle().WithR(-1).Node())},
		{name: "empty polygon", doc: NewDocument(256, 256).Append(NewPolygon().Node())},
		{name: "invalid node", doc: NewDocument(256, 256).Append(Node{})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := EncodeToString(tt.doc)
			require.Error(t, err)
		})
	}
	_, err := EncodeToString(NewDocument(256, 256).Append(NewPolygon().Node()))
	assert.ErrorIs(t, err, ErrPolygonPoints)
}

func TestEncodeStrokeBlackEmitted(t *testing.T) {
	t.Parallel()
	doc := NewDocument(256, 256).Append(
		NewCircle().WithR(5).WithStroke(NewStroke()).Node(),
	)
	got, err := EncodeToString(doc)
	require.NoError(t, err)
	assert.Contains(t, got, `stroke="#000000"`)
	assert.NotContains(t, got, "stroke-width")
}

func readGolden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "svg", name))
	require.NoError(t, err)
	return string(b)
}

func containsMinusZero(s string) bool {
	return strings.Contains(s, `="-0"`) || strings.Contains(s, `="-0 `) || strings.Contains(s, ` -0"`) || strings.Contains(s, ` -0 `)
}
