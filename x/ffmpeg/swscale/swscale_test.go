package swscale

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestYUV420ToRGB24MatchesFFmpeg(t *testing.T) {
	bin := test.Need(t, "ffmpeg")
	const w, h = 16, 16
	src, err := avutil.NewVideo(avutil.PixYUV420P, w, h)
	require.NoError(t, err)
	for y := 0; y < h; y++ {
		row := src.PlaneRow(0, y, w)
		for x := 0; x < w; x++ {
			row[x] = byte(16 + (x*13+y*7)%220)
		}
	}
	cw, ch := avutil.ChromaWidth(avutil.PixYUV420P, w), avutil.ChromaHeight(avutil.PixYUV420P, h)
	for y := 0; y < ch; y++ {
		u := src.PlaneRow(1, y, cw)
		v := src.PlaneRow(2, y, cw)
		for x := 0; x < cw; x++ {
			u[x] = byte(32 + (x*17+y*5)%200)
			v[x] = byte(48 + (x*11+y*19)%180)
		}
	}

	dir := t.TempDir()
	raw := filepath.Join(dir, "in.yuv")
	require.NoError(t, os.WriteFile(raw, tightYUV(src), 0o644))
	out := filepath.Join(dir, "out.rgb")
	cmd := exec.CommandContext(t.Context(), bin,
		"-hide_banner", "-cpuflags", "0", "-y",
		"-f", "rawvideo", "-pix_fmt", "yuv420p", "-s", "16x16", "-i", raw,
		"-sws_flags", "bicubic",
		"-pix_fmt", "rgb24", "-f", "rawvideo", out,
	)
	log, err := cmd.CombinedOutput()
	require.NoError(t, err, string(log))
	want, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Len(t, want, w*h*3)

	dst, err := avutil.NewVideo(avutil.PixRGB24, w, h)
	require.NoError(t, err)
	ctx, err := New(w, h, avutil.PixYUV420P, w, h, avutil.PixRGB24, AccurateRnd)
	require.NoError(t, err)
	require.NoError(t, ctx.Scale(dst, src))
	got := tightRGB(dst)
	require.Equal(t, want, got)
}

func tightYUV(frame *avutil.Frame) []byte {
	var out []byte
	for plane := 0; plane < 3; plane++ {
		width := frame.Width
		height := frame.Height
		if plane > 0 {
			width = avutil.ChromaWidth(frame.Format, frame.Width)
			height = avutil.ChromaHeight(frame.Format, frame.Height)
		}
		for y := 0; y < height; y++ {
			out = append(out, frame.PlaneRow(plane, y, width)...)
		}
	}
	return out
}

func tightRGB(frame *avutil.Frame) []byte {
	var out []byte
	for y := 0; y < frame.Height; y++ {
		out = append(out, frame.PlaneRow(0, y, frame.Width*3)...)
	}
	return out
}
