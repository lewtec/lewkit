package svg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathRoundTrip(t *testing.T) {
	p := NewPath().MoveTo(40, 40).LineTo(120, 40).LineTo(80, 100).Close()
	doc := NewDocument(256, 256).Append(p.Node())
	s, err := EncodeToString(doc)
	require.NoError(t, err)
	got, err := Parse(strings.NewReader(s))
	require.NoError(t, err)
	gp, ok := got.Children()[0].Path()
	require.True(t, ok)
	require.Len(t, gp.Commands(), 4)
}

func TestParseRelativeAndSmooth(t *testing.T) {
	doc, err := Parse(strings.NewReader(
		`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"><path d="m10 10 l10 0 c0 10 10 10 10 0 s10-10 10 0 z"/></svg>`,
	))
	require.NoError(t, err)
	p, ok := doc.Children()[0].Path()
	require.True(t, ok)
	cmds := p.Commands()
	require.Equal(t, CmdMove, cmds[0].Kind)
	require.Equal(t, 10.0, cmds[0].X)
	require.Equal(t, 10.0, cmds[0].Y)
	require.Equal(t, CmdLine, cmds[1].Kind)
	require.Equal(t, 20.0, cmds[1].X)
	require.Equal(t, 10.0, cmds[1].Y)
	require.Equal(t, CmdCubic, cmds[2].Kind)
	require.Equal(t, 20.0, cmds[2].X1)
	require.Equal(t, 20.0, cmds[2].Y1)
	require.Equal(t, 30.0, cmds[2].X2)
	require.Equal(t, 20.0, cmds[2].Y2)
	require.Equal(t, 30.0, cmds[2].X)
	require.Equal(t, 10.0, cmds[2].Y)
	require.Equal(t, CmdCubic, cmds[3].Kind)
	require.Equal(t, 30.0, cmds[3].X1)
	require.Equal(t, 0.0, cmds[3].Y1)
	require.Equal(t, 40.0, cmds[3].X2)
	require.Equal(t, 0.0, cmds[3].Y2)
	require.Equal(t, 40.0, cmds[3].X)
	require.Equal(t, 10.0, cmds[3].Y)
	require.Equal(t, CmdClose, cmds[4].Kind)
}

func TestParseQuadraticAndSmooth(t *testing.T) {
	doc, err := Parse(strings.NewReader(
		`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"><path d="M0 0 Q10 10 20 0 T40 0"/></svg>`,
	))
	require.NoError(t, err)
	p, ok := doc.Children()[0].Path()
	require.True(t, ok)
	cmds := p.Commands()
	require.Len(t, cmds, 3)
	require.Equal(t, CmdCubic, cmds[1].Kind)
	require.InDelta(t, 20.0/3.0, cmds[1].X1, 1e-9)
	require.InDelta(t, 20.0/3.0, cmds[1].Y1, 1e-9)
	require.InDelta(t, 40.0/3.0, cmds[1].X2, 1e-9)
	require.InDelta(t, 20.0/3.0, cmds[1].Y2, 1e-9)
	require.InDelta(t, 20.0, cmds[1].X, 1e-9)
	require.InDelta(t, 0.0, cmds[1].Y, 1e-9)
	require.Equal(t, CmdCubic, cmds[2].Kind)
	require.InDelta(t, 40.0, cmds[2].X, 1e-9)
	require.InDelta(t, 0.0, cmds[2].Y, 1e-9)
	require.InDelta(t, 80.0/3.0, cmds[2].X1, 1e-9)
	require.InDelta(t, -20.0/3.0, cmds[2].Y1, 1e-9)
}

func TestParseArcLowersToCubics(t *testing.T) {
	for _, d := range []string{
		`M0 0 A10 10 0 0 1 10 10`,
		`M0 0 A10 10 0 0010 10`,
		`M0 0 a10 10 0 0 1 10 10`,
	} {
		doc, err := Parse(strings.NewReader(
			`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"><path d="` + d + `"/></svg>`,
		))
		require.NoError(t, err)
		p, ok := doc.Children()[0].Path()
		require.True(t, ok)
		cmds := p.Commands()
		require.GreaterOrEqual(t, len(cmds), 2)
		last := cmds[len(cmds)-1]
		require.Equal(t, CmdCubic, last.Kind)
		require.InDelta(t, 10.0, last.X, 1e-6)
		require.InDelta(t, 10.0, last.Y, 1e-6)
	}
}

func TestParseArcFlag(t *testing.T) {
	_, err := Parse(strings.NewReader(
		`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256"><path d="M0 0 A10 10 0 2 1 10 10"/></svg>`,
	))
	require.Error(t, err)
}
