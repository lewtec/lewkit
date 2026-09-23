package experiments

import (
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/test"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/require"
)

func BenchmarkScrollPresent(b *testing.B) {
	const width, height = 1024, 1024
	ctx := b.Context()
	host, err := window.Open(ctx, window.Config{Title: "lewkit scroll", Width: width, Height: height})
	if err != nil {
		b.Skip(err)
	}
	test.CloseOnCleanup(b, host)
	surfacer, ok := host.(window.Surfacer)
	if !ok {
		b.Skip("window has no vulkan surface")
	}
	surface := surfacer.Surface()
	screen, err := vulkan.OpenNative(ctx, surface.Kind, surface.A, surface.B, width, height)
	if err != nil {
		b.Skip(err)
	}
	test.CloseOnCleanup(b, screen)

	marquee, err := gui.NewMarquee()
	require.NoError(b, err)
	size := image.Pt(width, height)
	marquee.Update(window.Resize{Size: size})
	picture, err := gui.NewPicture()
	require.NoError(b, err)

	frame := func(elapsed time.Duration) {
		marquee.Update(gui.TickMsg{Elapsed: elapsed, Size: size, Period: window.DefaultFramePeriod})
		require.NoError(b, picture.Show(ctx, screen, marquee.View(), gui.Size{Width: width, Height: height}))
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
