package avcodec

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestH264IPCMRoundTrip(t *testing.T) {
	frame, err := avutil.NewVideo(avutil.PixYUV420P, 18, 16)
	require.NoError(t, err)
	for y := 0; y < frame.Height; y++ {
		row := frame.Data[0][y*frame.Linesize[0]:]
		for x := 0; x < frame.Width; x++ {
			row[x] = byte(x*3 + y*5)
			if x == 2 && y == 1 {
				row[x] = 0
				row[x+1] = 0
				row[x+2] = 1
			}
		}
	}
	for y := 0; y < (frame.Height+1)/2; y++ {
		for x := 0; x < (frame.Width+1)/2; x++ {
			frame.Data[1][y*frame.Linesize[1]+x] = byte(40 + x)
			frame.Data[2][y*frame.Linesize[2]+x] = byte(80 + y)
		}
	}
	enc := newH264(Parameters{ID: IDH264, Width: 18, Height: 16, Format: avutil.PixYUV420P})
	pkts, err := enc.Encode(frame)
	require.NoError(t, err)
	require.Len(t, pkts, 1)

	dec := newH264(Parameters{ID: IDH264})
	got, err := dec.Decode(pkts[0])
	require.NoError(t, err)
	require.Equal(t, plane(frame, 0), plane(got[0], 0))
	require.Equal(t, plane(frame, 1), plane(got[0], 1))
	require.Equal(t, plane(frame, 2), plane(got[0], 2))

	ff := test.Need(t, "ffmpeg")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-f", "h264", "-i", "pipe:0", "-f", "rawvideo", "-pix_fmt", "yuv420p", "-")
	cmd.Stdin = bytes.NewReader(pkts[0].Data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	require.NoError(t, cmd.Run(), stderr.String())
	raw := stdout.Bytes()
	require.Equal(t, 18*16+9*8+9*8, len(raw))
	require.Equal(t, tight(frame, 0), raw[:18*16])
	require.Equal(t, tight(frame, 1), raw[18*16:18*16+9*8])
	require.Equal(t, tight(frame, 2), raw[18*16+9*8:])
}

func TestAddClip8MatchesScalar(t *testing.T) {
	dst := []byte{0, 1, 250, 255, 10, 20, 30, 40, 5, 6}
	residual := []int16{1, -1, 20, 5, -100, 300, 0, -40, 1, 1}
	scalar := append([]byte(nil), dst...)
	avutil.SetScalar(true)
	addClip8(scalar, residual)
	avutil.SetScalar(false)
	addClip8(dst, residual)
	require.Equal(t, scalar, dst)
	block := [16]int16{32, -8, 4, 0, 2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}
	pix := []byte{10, 20, 30, 40, 11, 21, 31, 41, 12, 22, 32, 42, 13, 23, 33, 43}
	h264IDCTAdd(pix, 4, &block)
	require.Equal(t, [16]int16{}, block)
	require.NotEqual(t, []byte{10, 20, 30, 40}, pix[:4])
}

func plane(frame *avutil.Frame, p int) []byte {
	h := frame.Height
	if p > 0 {
		h = (h + 1) / 2
	}
	return frame.Data[p][:h*frame.Linesize[p]]
}

func tight(frame *avutil.Frame, p int) []byte {
	w, h := frame.Width, frame.Height
	if p > 0 {
		w, h = (w+1)/2, (h+1)/2
	}
	out := make([]byte, 0, w*h)
	for y := 0; y < h; y++ {
		out = append(out, frame.Data[p][y*frame.Linesize[p]:y*frame.Linesize[p]+w]...)
	}
	return out
}
