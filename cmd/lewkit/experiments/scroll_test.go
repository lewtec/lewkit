package experiments

import (
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/test"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/require"
)

func BenchmarkScroll(b *testing.B) {
	const (
		width       = 1024
		height      = 1024
		evaluatorID = "ndeval_vulkan"
	)
	ctx := b.Context()
	if _, err := vulkan.List(ctx); err != nil {
		b.Skip(err)
	}
	handles, err := driver.List[ndarray.Evaluator](ctx)
	require.NoError(b, err)
	var evaluator ndarray.Evaluator
	for _, handle := range handles {
		if handle.ID != evaluatorID {
			continue
		}
		evaluator, err = handle.Open(ctx)
		require.NoError(b, err)
		break
	}
	require.NotNil(b, evaluator)
	test.CloseOnCleanup(b, evaluator)

	marquee, err := gui.NewMarquee()
	require.NoError(b, err)
	size := image.Pt(width, height)
	marquee.Update(window.Resize{Size: size})
	picture, err := gui.NewPicture()
	require.NoError(b, err)
	pixels := make([]uint8, width*height*4)

	frame := func(elapsed time.Duration) {
		marquee.Update(gui.TickMsg{Elapsed: elapsed, Size: size, Period: window.DefaultFramePeriod})
		view, err := picture.Render(marquee.View(), gui.Size{Width: width, Height: height})
		require.NoError(b, err)
		require.NoError(b, view.Eval(ctx, evaluator, pixels))
	}
	frame(0)
	b.ReportAllocs()
	b.SetBytes(int64(len(pixels)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		frame(time.Duration(i+1) * window.DefaultFramePeriod)
	}
}
