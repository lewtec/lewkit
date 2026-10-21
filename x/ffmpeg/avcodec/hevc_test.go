package avcodec

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestHEVCPCMRoundTrip(t *testing.T) {
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
	for y := 0; y < frame.Height/2; y++ {
		for x := 0; x < frame.Width/2; x++ {
			frame.Data[1][y*frame.Linesize[1]+x] = byte(40 + x)
			frame.Data[2][y*frame.Linesize[2]+x] = byte(80 + y)
		}
	}
	enc := newHEVC(Parameters{ID: IDHEVC, Width: 18, Height: 16, Format: avutil.PixYUV420P})
	pkts, err := enc.Encode(frame)
	require.NoError(t, err)
	require.Len(t, pkts, 1)

	dec := newHEVC(Parameters{ID: IDHEVC})
	got, err := dec.Decode(pkts[0])
	require.NoError(t, err)
	require.Equal(t, plane(frame, 0), plane(got[0], 0))
	require.Equal(t, plane(frame, 1), plane(got[0], 1))
	require.Equal(t, plane(frame, 2), plane(got[0], 2))

	ff := test.Need(t, "ffmpeg")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-f", "hevc", "-i", "pipe:0", "-f", "rawvideo", "-pix_fmt", "yuv420p", "-")
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

func TestHEVCCopy16MatchesScalar(t *testing.T) {
	src := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 99}
	scalar := make([]byte, 16)
	avutil.SetScalar(true)
	hevcCopy16(scalar, src)
	avutil.SetScalar(false)
	dst := make([]byte, 16)
	hevcCopy16(dst, src)
	require.Equal(t, src[:16], scalar)
	require.Equal(t, scalar, dst)
}
