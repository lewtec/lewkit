package avcodec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
	"math"
	"os"
	"os/exec"
	"testing"
)

// TestVorbisEncodeMatchesFFmpeg encodes 4410 samples (0.1s at 44100 Hz) of a
// 440 Hz sine. FFmpeg's native vorbis encoder rejects -ac 1 ("only supports 2
// channels"), so the reference is the same sine duplicated to stereo:
//
//	ffmpeg -hide_banner -nostdin -loglevel error -f s16le -ar 44100 -ac 2 -i <file> -strict experimental -c:a vorbis -f ogg <out.ogg>
//
// Ogg pages and the muxer vendor string are ignored; audio packet payloads are
// compared bit-exactly.
func TestVorbisEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const (
		rate    = 44100
		samples = 4410
	)
	pcm := make([]int16, samples)
	for i := range pcm {
		pcm[i] = int16(math.Round(math.Sin(2*math.Pi*440*float64(i)/float64(rate)) * 16000))
	}
	got, err := VorbisEncodeMono(pcm, rate)
	require.NoError(t, err)
	if len(got) == 0 {
		require.Fail(t, "empty vorbis packets")
	}

	raw := make([]byte, samples*4)
	for i, s := range pcm {
		binary.LittleEndian.PutUint16(raw[i*4:], uint16(s))
		binary.LittleEndian.PutUint16(raw[i*4+2:], uint16(s))
	}
	in, err := os.CreateTemp("", "vorbis-*.pcm")
	require.NoError(t, err)
	defer os.Remove(in.Name())
	_, err = in.Write(raw)
	require.NoError(t, err)
	in.Close()
	out, err := os.CreateTemp("", "vorbis-*.ogg")
	require.NoError(t, err)
	out.Close()
	defer os.Remove(out.Name())

	cmd := exec.Command(ff, "-y", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "2", "-i", in.Name(),
		"-strict", "experimental", "-c:a", "vorbis", "-f", "ogg", out.Name(),
	)
	if msg, err := cmd.CombinedOutput(); err != nil {
		require.FailNow(t, fmt.Sprintf("ffmpeg: %v %s", err, msg))
	}
	ogg, err := os.ReadFile(out.Name())
	require.NoError(t, err)
	pkts, err := oggPackets(ogg)
	require.NoError(t, err)
	var want []byte
	for _, p := range pkts {
		if isVorbisHeader(p) {
			continue
		}
		want = append(want, p...)
	}
	if !bytes.Equal(got, want) {
		n := len(got)
		if len(want) < n {
			n = len(want)
		}
		off := n
		for i := 0; i < n; i++ {
			if got[i] != want[i] {
				off = i
				break
			}
		}
		require.FailNow(t, fmt.Sprintf("vorbis bytes mismatch got %d want %d first diff %d", len(got), len(want), off))
	}
}

func isVorbisHeader(p []byte) bool {
	return len(p) >= 7 && (p[0] == 1 || p[0] == 3 || p[0] == 5) && string(p[1:7]) == "vorbis"
}

func oggPackets(b []byte) ([][]byte, error) {
	var pkts [][]byte
	var cur []byte
	for len(b) > 0 {
		if len(b) < 27 || string(b[:4]) != "OggS" {
			return nil, fmt.Errorf("ogg sync")
		}
		nseg := int(b[26])
		if len(b) < 27+nseg {
			return nil, fmt.Errorf("ogg segments")
		}
		segs := b[27 : 27+nseg]
		body := 27 + nseg
		size := 0
		for _, s := range segs {
			size += int(s)
		}
		if len(b) < body+size {
			return nil, fmt.Errorf("ogg body")
		}
		data := b[body : body+size]
		off := 0
		for _, s := range segs {
			cur = append(cur, data[off:off+int(s)]...)
			off += int(s)
			if s < 255 {
				pkts = append(pkts, cur)
				cur = nil
			}
		}
		b = b[body+size:]
	}
	if len(cur) > 0 {
		pkts = append(pkts, cur)
	}
	return pkts, nil
}
