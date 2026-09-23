package experiments

import (
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/vulkanwindow"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/require"
)

func BenchmarkScrollPresent(b *testing.B) {
	const width, height = 1024, 1024
	ctx := b.Context()
	host, err := vulkanwindow.Open(ctx, window.Config{Title: "lewkit scroll", Width: width, Height: height})
	if err != nil {
		b.Skip(err)
	}
	b.Cleanup(func() { require.NoError(b, host.Close()) })

	marquee, err := gui.NewMarquee()
	require.NoError(b, err)
	size := image.Pt(width, height)
	marquee.Update(window.Resize{Size: size})
	picture, err := gui.NewPicture()
	require.NoError(b, err)

	frame := func(elapsed time.Duration) {
		marquee.Update(gui.TickMsg{Elapsed: elapsed, Size: size, Period: window.DefaultFramePeriod})
		view, err := picture.Render(marquee.View(), gui.Size{Width: width, Height: height})
		require.NoError(b, err)
		require.NoError(b, host.Paint(ctx, view))
	}
	frame(0)
	b.ReportAllocs()
	b.SetBytes(int64(width * height * 4))
	var i int
	for b.Loop() {
		i++
		frame(time.Duration(i) * window.DefaultFramePeriod)
	}
}
