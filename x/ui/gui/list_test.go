package gui

import (
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/driver/present"
	"github.com/lewtec/lewkit/x/driver/window"
	_ "github.com/lewtec/lewkit/x/driver/window/mem"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

type surfaceHost struct{ window.Window }

func (surfaceHost) Surface() window.Surface {
	return window.Surface{Kind: window.SurfaceUIView, A: 1}
}

type captureScreen struct {
	under []byte
}

func (s *captureScreen) Draw(_ context.Context, _, under, _ []byte, _, _ int) error {
	s.under = append([]byte(nil), under...)
	return nil
}

func (*captureScreen) Adopt(uintptr, int, int) error { return nil }
func (*captureScreen) Close() error                  { return nil }

// markProgramEval fills every backdrop byte with 7. The CPU tape would not.
type markProgramEval struct{}

func (markProgramEval) Program(context.Context, *ndarray.Kernel) (ndarray.Program, error) {
	return markFill{}, nil
}

func (markProgramEval) Close() error { return nil }

type markFill struct{}

func (markFill) Eval(_ context.Context, output []byte) error {
	for i := range output {
		output[i] = 7
	}
	return nil
}

func (markFill) Close() error { return nil }

func TestListBackdropUsesHeldEvaluator(t *testing.T) {
	shape := ndarray.Shape{1, 1, 4}
	source := ndarray.Coord(1, shape).Cast[float32]()
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.raster = source
	host, err := window.Open(t.Context(), window.Config{Width: 2, Height: 2})
	require.NoError(t, err)
	t.Cleanup(func() { _ = host.Close() })
	screen := &captureScreen{}
	display := bridgeDisplay{
		Window: surfaceHost{host},
		screen: screen,
		raster: &rasterHold{eval: markProgramEval{}},
	}
	require.NoError(t, display.presentList(t.Context(), picture))
	require.Len(t, screen.under, 2*2*4)
	require.Equal(t, uint8(7), screen.under[0])
	require.Equal(t, uint8(7), screen.under[len(screen.under)-1])
}

var _ present.Screen = (*captureScreen)(nil)
var _ ndarray.Evaluator = markProgramEval{}
