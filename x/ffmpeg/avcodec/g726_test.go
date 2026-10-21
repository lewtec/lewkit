package avcodec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
	"os/exec"
	"testing"
)

func TestG726LEMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=0.04",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	samples := make([]int16, len(pcm)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got, err := G726LEEncode(samples, 4)
	require.NoError(t, err)
	enc := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", "pipe:0",
		"-c:a", "g726le", "-f", "g726le", "-",
	)
	enc.Stdin = bytes.NewReader(pcm)
	want, err := enc.Output()
	require.NoError(t, err)
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("g726le bytes mismatch\n got %x\nwant %x", got, want))
	}
	dec := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "g726le", "-i", "pipe:0", "-f", "s16le", "-",
	)
	dec.Stdin = bytes.NewReader(want)
	raw, err := dec.Output()
	require.NoError(t, err)
	decoded, err := G726LEDecode(got, 4)
	require.NoError(t, err)
	if len(raw) != len(decoded)*2 {
		require.FailNow(t, fmt.Sprintf("decoded samples %d ffmpeg %d", len(decoded), len(raw)/2))
	}
	for i, sample := range decoded {
		wantSample := int16(binary.LittleEndian.Uint16(raw[i*2:]))
		if sample != wantSample {
			require.FailNow(t, fmt.Sprintf("sample %d got %d want %d", i, sample, wantSample))
		}
	}
}

func TestG726MatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=0.04",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	if len(pcm) != 320*2 {
		require.FailNow(t, fmt.Sprintf("pcm length %d", len(pcm)))
	}
	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got, err := G726Encode(samples, 4)
	require.NoError(t, err)
	enc := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", "pipe:0",
		"-c:a", "g726", "-f", "g726", "-",
	)
	enc.Stdin = bytes.NewReader(pcm)
	want, err := enc.Output()
	require.NoError(t, err)
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("g726 bytes mismatch\n got %x\nwant %x", got, want))
	}
	dec := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "g726", "-i", "pipe:0", "-f", "s16le", "-",
	)
	dec.Stdin = bytes.NewReader(want)
	raw, err := dec.Output()
	require.NoError(t, err)
	decoded, err := G726Decode(got, 4)
	require.NoError(t, err)
	if len(raw) != len(decoded)*2 {
		require.FailNow(t, fmt.Sprintf("decoded samples %d ffmpeg %d", len(decoded), len(raw)/2))
	}
	for i, sample := range decoded {
		wantSample := int16(binary.LittleEndian.Uint16(raw[i*2:]))
		if sample != wantSample {
			require.FailNow(t, fmt.Sprintf("sample %d got %d want %d", i, sample, wantSample))
		}
	}
}
