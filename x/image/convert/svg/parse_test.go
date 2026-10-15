package svg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRoundTripGoldens(t *testing.T) {
	t.Parallel()
	dir := filepath.Join("testdata", "svg")
	ents, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, ent := range ents {
		if !strings.HasSuffix(ent.Name(), ".svg") {
			continue
		}
		t.Run(ent.Name(), func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(filepath.Join(dir, ent.Name()))
			require.NoError(t, err)
			doc, err := Parse(strings.NewReader(string(raw)))
			require.NoError(t, err)
			got, err := EncodeToString(doc)
			require.NoError(t, err)
			assert.Equal(t, string(raw), got)
		})
	}
}

func TestParseFile(t *testing.T) {
	t.Parallel()
	doc, err := ParseFile(filepath.Join("testdata", "svg", "empty.svg"))
	require.NoError(t, err)
	assert.Equal(t, 256.0, doc.Width())
	assert.Equal(t, 256.0, doc.Height())
}

func TestParseRXCopyWhenOmitted(t *testing.T) {
	t.Parallel()
	doc, err := Parse(strings.NewReader(`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256">
  <rect width="10" height="10" rx="5"/>
</svg>`))
	require.NoError(t, err)
	n := doc.Children()
	require.Len(t, n, 1)
	r, ok := n[0].Rect()
	require.True(t, ok)
	assert.Equal(t, 5.0, r.RX())
	assert.Equal(t, 5.0, r.RY())
}

func TestParseRXBothPresentKeepsZeroRY(t *testing.T) {
	t.Parallel()
	doc, err := ParseFile(filepath.Join("testdata", "svg", "rx-only.svg"))
	require.NoError(t, err)
	r, ok := doc.Children()[0].Rect()
	require.True(t, ok)
	assert.Equal(t, 5.0, r.RX())
	assert.Equal(t, 0.0, r.RY())
}

func TestParseNoXMLNS(t *testing.T) {
	t.Parallel()
	doc, err := Parse(strings.NewReader(`<svg width="256" height="256"/>`))
	require.NoError(t, err)
	got, err := EncodeToString(doc)
	require.NoError(t, err)
	assert.Equal(t, readGolden(t, "empty.svg"), got)
}

func TestParsePxUnits(t *testing.T) {
	t.Parallel()
	doc, err := Parse(strings.NewReader(`<svg width="256px" height="256px"/>`))
	require.NoError(t, err)
	assert.Equal(t, 256.0, doc.Width())
	assert.Equal(t, 256.0, doc.Height())
}

func TestParseRRGGBBAATimesOpacity(t *testing.T) {
	t.Parallel()
	doc, err := Parse(strings.NewReader(
		`<svg width="256" height="256"><circle r="1" fill="#FF000080" fill-opacity="0.5"/></svg>`,
	))
	require.NoError(t, err)
	c, ok := doc.Children()[0].Circle()
	require.True(t, ok)
	col, present := c.Fill()
	assert.True(t, present)
	assert.Equal(t, uint8(255), col.R)
	assert.Equal(t, uint8(0), col.G)
	assert.Equal(t, uint8(0), col.B)
	want := PremultiplyU8(0x80, op8FromUnit(0.5))
	assert.Equal(t, want, op8FromUnit(c.FillOpacity()))
}

func TestParseRejects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		xml  string
	}{
		{name: "fill on g", xml: `<svg width="256" height="256"><g fill="#FF0000"/></svg>`},
		{name: "style", xml: `<svg width="256" height="256" style="x"/></svg>`},
		{name: "transform", xml: `<svg width="256" height="256"><circle r="1" transform="scale(2)"/></svg>`},
		{name: "named color", xml: `<svg width="256" height="256"><circle r="1" fill="red"/></svg>`},
		{name: "rgb()", xml: `<svg width="256" height="256"><circle r="1" fill="rgb(0,0,0)"/></svg>`},
		{name: "winding", xml: `<svg width="256" height="256"><polygon points="0,0 1,0 0,1" fill-rule="winding"/></svg>`},
		{name: "text", xml: `<svg width="256" height="256">hello</svg>`},
		{name: "path arc", xml: `<svg width="256" height="256"><path d="M0 0 A10 10 0 0 1 10 10"/></svg>`},
		{name: "em unit", xml: `<svg width="10em" height="256"/></svg>`},
		{name: "non-integer canvas", xml: `<svg width="256.5" height="256"/>`},
		{name: "negative r", xml: `<svg width="256" height="256"><circle r="-1"/></svg>`},
		{name: "too many points", xml: tooManyPointsXML()},
		{name: "stroke width only", xml: `<svg width="256" height="256"><circle r="1" stroke-width="2"/></svg>`},
		{name: "doctype", xml: `<!DOCTYPE svg><svg width="256" height="256"/>`},
		{name: "objectBoundingBox", xml: `<svg width="64" height="16"><defs><linearGradient id="lg0"><stop offset="0" stop-color="#FF0000"/><stop offset="1" stop-color="#000000"/></linearGradient></defs><rect width="64" height="16" fill="url(#lg0)"/></svg>`},
		{name: "unknown gradient", xml: `<svg width="64" height="16"><rect width="64" height="16" fill="url(#lg0)"/></svg>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(strings.NewReader(tt.xml))
			require.Error(t, err)
		})
	}
}

func TestParseCommentsAndSpace(t *testing.T) {
	t.Parallel()
	doc, err := Parse(strings.NewReader(`
		<!-- logo -->
		<svg width="256" height="256">
		  <!-- inner -->
		  <circle r="1"/>
		</svg>
	`))
	require.NoError(t, err)
	require.Len(t, doc.Children(), 1)
}

func TestParsePolygonPointsGrammar(t *testing.T) {
	t.Parallel()
	doc, err := Parse(strings.NewReader(
		`<svg width="256" height="256"><polygon points="0,0, 10 0,0,10"/></svg>`,
	))
	require.NoError(t, err)
	p, ok := doc.Children()[0].Polygon()
	require.True(t, ok)
	assert.Equal(t, [][2]float64{{0, 0}, {10, 0}, {0, 10}}, p.Points())
}

func TestParseOddPoints(t *testing.T) {
	t.Parallel()
	_, err := Parse(strings.NewReader(
		`<svg width="256" height="256"><polygon points="0,0 1,0 2"/></svg>`,
	))
	require.Error(t, err)
}

func TestParseEmptyPolygonRejectedOnEncode(t *testing.T) {
	t.Parallel()
	doc, err := Parse(strings.NewReader(`<svg width="256" height="256"><polygon/></svg>`))
	require.NoError(t, err)
	_, err = EncodeToString(doc)
	assert.ErrorIs(t, err, ErrPolygonPoints)
}

func tooManyPointsXML() string {
	var b strings.Builder
	b.WriteString(`<svg width="256" height="256"><polygon points="`)
	for i := 0; i < 1025; i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("0,0")
	}
	b.WriteString(`"/></svg>`)
	return b.String()
}
