package experiments

import (
	"context"
	"fmt"
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/require"
)

const (
	scrollBenchWidth  = 1024
	scrollBenchHeight = 1024
	vulkanEvaluatorID = "ndeval_vulkan"
)

func BenchmarkScroll(b *testing.B) {
	ctx := b.Context()
	if _, err := vulkan.List(ctx); err != nil {
		b.Skip(err)
	}
	evaluator, err := openVulkanEvaluator(ctx)
	require.NoError(b, err)
	b.Cleanup(func() { require.NoError(b, evaluator.Close()) })

	marquee, err := gui.NewMarquee()
	require.NoError(b, err)
	size := image.Pt(scrollBenchWidth, scrollBenchHeight)
	marquee.Update(window.Resize{Size: size})
	picture, err := gui.NewPicture()
	require.NoError(b, err)
	pixels := make([]uint8, scrollBenchWidth*scrollBenchHeight*4)

	frame := func(elapsed time.Duration) {
		marquee.Update(gui.TickMsg{Elapsed: elapsed, Size: size, Period: window.DefaultFramePeriod})
		view, err := picture.Render(marquee.View(), gui.Size{Width: scrollBenchWidth, Height: scrollBenchHeight})
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

func openVulkanEvaluator(ctx context.Context) (ndarray.Evaluator, error) {
	handles, err := driver.List[ndarray.Evaluator](ctx)
	if err != nil {
		return nil, err
	}
	for _, handle := range handles {
		if handle.ID != vulkanEvaluatorID {
			continue
		}
		return handle.Open(ctx)
	}
	return nil, fmt.Errorf("%w: %s", driver.ErrUnavailable, vulkanEvaluatorID)
}
