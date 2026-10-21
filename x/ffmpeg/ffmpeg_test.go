package ffmpeg

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func runFF(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(t.Context(), args, bytes.NewReader(nil), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func systemFF(t *testing.T, args ...string) []byte {
	t.Helper()
	bin := test.Need(t, "ffmpeg")
	cmd := exec.Command(bin, append([]string{"-hide_banner", "-nostdin", "-loglevel", "error", "-y", "-cpuflags", "0"}, args...)...)
	out, err := cmd.Output()
	require.NoError(t, err)
	return out
}

func TestNoArgs(t *testing.T) {
	code, stdout, stderr := runFF(t)
	require.Equal(t, 1, code)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "Hyper fast Audio and Video encoder\nusage: ffmpeg [options] [[infile options] -i infile]... {[outfile options] outfile}...\n\nUse -h to get full help or, even better, run 'man ffmpeg'\n")
}

func TestUnrecognizedOption(t *testing.T) {
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-foo")
	require.Equal(t, 1, code)
	require.Empty(t, stdout)
	require.Equal(t, "Unrecognized option 'foo'.\nError splitting the argument list: Option not found\n", stderr)
}

func TestMissingArgument(t *testing.T) {
	code, _, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-i")
	require.Equal(t, 1, code)
	require.Equal(t, "Missing argument for option 'i'.\nError splitting the argument list: Invalid argument\n", stderr)
}

func TestHelpAndVersion(t *testing.T) {
	code, stdout, stderr := runFF(t, "-h")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "usage: ffmpeg [options]")
	require.Contains(t, stderr, "ffmpeg version "+Version)
	code, stdout, stderr = runFF(t, "-version")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "ffmpeg version "+Version)
	require.Contains(t, stdout, "--enable-decoder=rawvideo")
	require.Contains(t, stdout, "--enable-filter=testsrc")
	require.Empty(t, stderr)
	code, stdout, _ = runFF(t, "-hide_banner", "-formats")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "wav")
	require.Contains(t, stdout, "framecrc")
	code, stdout, _ = runFF(t, "-hide_banner", "-codecs")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "rawvideo")
	require.Contains(t, stdout, "pcm_s16le")
	code, stdout, _ = runFF(t, "-hide_banner", "-filters")
	require.Equal(t, 0, code)
	require.Contains(t, stdout, "sine")
}

func TestNoOutput(t *testing.T) {
	code, _, stderr := runFF(t, "-loglevel", "error", "-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04")
	require.Equal(t, 1, code)
	require.Equal(t, "At least one output file must be specified\n", stderr)
}

func TestOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.rgb")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
	code, _, stderr := runFF(t, "-loglevel", "error", "-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04", "-frames:v", "1", "-f", "rawvideo", path)
	require.Equal(t, 1, code)
	require.Equal(t, "File '"+path+"' already exists. Exiting.\n", stderr)
}

func TestTestSrcMatchesFFmpeg(t *testing.T) {
	want := systemFF(t, "-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04", "-frames:v", "1", "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0", "-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04", "-frames:v", "1", "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	require.Equal(t, 0, code, stderr)
	require.Equal(t, want, []byte(stdout))
	require.Len(t, stdout, 768)
}

func TestFrameCRCMatchesFFmpeg(t *testing.T) {
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0", "-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04", "-frames:v", "1", "-f", "framecrc", "-")
	require.Equal(t, 0, code, stderr)
	require.Contains(t, stdout, "#software: "+Version+"\n")
	require.Contains(t, stdout, "#tb 0: 1/25\n")
	require.Contains(t, stdout, "#dimensions 0: 16x16\n")
	require.Contains(t, stdout, "#codec_id 0: rawvideo\n")
	require.Contains(t, stdout, "0,          0,          0,        1,      768, 0x887c60ad\n")
	code, stdout, stderr = runFF(t, "-hide_banner", "-loglevel", "error", "-bitexact", "-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04", "-frames:v", "1", "-f", "framecrc", "-")
	require.Equal(t, 0, code, stderr)
	require.NotContains(t, stdout, "software")
	require.Contains(t, stdout, "0x887c60ad")
}

func TestSineMatchesFFmpeg(t *testing.T) {
	spec := "sine=frequency=440:sample_rate=44100:duration=0.023219954648526078"
	want := systemFF(t, "-f", "lavfi", "-i", spec, "-frames:a", "1", "-f", "s16le", "-")
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0", "-f", "lavfi", "-i", spec, "-frames:a", "1", "-f", "s16le", "-")
	require.Equal(t, 0, code, stderr)
	require.Equal(t, want, []byte(stdout))
	require.Len(t, stdout, 2048)
}

func TestFilterComplex(t *testing.T) {
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0", "-filter_complex", "testsrc=size=16x16:rate=25:duration=0.04", "-frames:v", "1", "-f", "framecrc", "-")
	require.Equal(t, 0, code, stderr)
	require.Contains(t, stdout, "0x887c60ad")
}

func TestPNGRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "frame.png")
	code, _, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0", "-y", "-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04", "-frames:v", "1", path)
	require.Equal(t, 0, code, stderr)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "\x89PNG\r\n\x1a\n", string(body[:8]))
	got := systemFF(t, "-i", path, "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	want := systemFF(t, "-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04", "-frames:v", "1", "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	require.Equal(t, want, got)
}

func TestWAVAndFLAC(t *testing.T) {
	spec := "sine=frequency=440:sample_rate=8000:duration=0.05"
	want := systemFF(t, "-f", "lavfi", "-i", spec, "-f", "s16le", "-")
	dir := t.TempDir()
	wav := filepath.Join(dir, "tone.wav")
	flac := filepath.Join(dir, "tone.flac")
	code, _, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0", "-y", "-f", "lavfi", "-i", spec, "-c:a", "pcm_s16le", wav)
	require.Equal(t, 0, code, stderr)
	got := systemFF(t, "-i", wav, "-f", "s16le", "-")
	require.Equal(t, want, got)
	code, _, stderr = runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0", "-y", "-f", "lavfi", "-i", spec, "-c:a", "flac", flac)
	require.Equal(t, 0, code, stderr)
	body, err := os.ReadFile(flac)
	require.NoError(t, err)
	require.Equal(t, "fLaC", string(body[:4]))
	got = systemFF(t, "-i", flac, "-f", "s16le", "-")
	require.Equal(t, want, got)
}

func TestNegateVFlipCropMatchFFmpeg(t *testing.T) {
	for _, vf := range []string{"negate", "vflip", "crop=8:8:4:4", "hflip,negate"} {
		args := []string{"-f", "lavfi", "-i", "testsrc=size=32x32:rate=25:duration=0.04", "-frames:v", "1", "-vf", vf, "-f", "rawvideo", "-pix_fmt", "rgb24", "-"}
		want := systemFF(t, args...)
		code, stdout, stderr := runFF(t, append([]string{"-hide_banner", "-loglevel", "error", "-cpuflags", "0"}, args...)...)
		require.Equal(t, 0, code, vf+" "+stderr)
		require.Equal(t, want, []byte(stdout), vf)
	}
}

func TestTransposePadVolumeMatchFFmpeg(t *testing.T) {
	video := []string{"transpose=1", "pad=40:36:4:2:black"}
	for _, vf := range video {
		args := []string{"-f", "lavfi", "-i", "testsrc=size=32x24:rate=25:duration=0.04", "-frames:v", "1", "-vf", vf, "-f", "rawvideo", "-pix_fmt", "rgb24", "-"}
		want := systemFF(t, args...)
		code, stdout, stderr := runFF(t, append([]string{"-hide_banner", "-loglevel", "error", "-cpuflags", "0"}, args...)...)
		require.Equal(t, 0, code, vf+" "+stderr)
		require.Equal(t, want, []byte(stdout), vf)
	}
	spec := "sine=frequency=440:sample_rate=8000:duration=0.05"
	args := []string{"-f", "lavfi", "-i", spec, "-af", "volume=0.5", "-f", "s16le", "-"}
	want := systemFF(t, args...)
	code, stdout, stderr := runFF(t, append([]string{"-hide_banner", "-loglevel", "error", "-cpuflags", "0"}, args...)...)
	require.Equal(t, 0, code, stderr)
	require.Equal(t, want, []byte(stdout))
}

func TestCopyFrameCRC(t *testing.T) {
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0", "-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04", "-frames:v", "1", "-c", "copy", "-f", "framecrc", "-")
	require.Equal(t, 0, code, stderr)
	require.Contains(t, stdout, "0x887c60ad")
}

func TestHFlipMatchesFFmpeg(t *testing.T) {
	args := []string{"-f", "lavfi", "-i", "testsrc=size=32x32:rate=25:duration=0.04", "-frames:v", "1", "-vf", "hflip", "-f", "rawvideo", "-pix_fmt", "rgb24", "-"}
	want := systemFF(t, args...)
	code, stdout, stderr := runFF(t, append([]string{"-hide_banner", "-loglevel", "error", "-cpuflags", "0"}, args...)...)
	require.Equal(t, 0, code, stderr)
	require.Equal(t, want, []byte(stdout))
}

func TestAVIAndMOV(t *testing.T) {
	dir := t.TempDir()
	avi := filepath.Join(dir, "c.avi")
	mov := filepath.Join(dir, "c.mov")
	spec := "color=c=red:s=16x16:r=25:d=0.08"
	code, _, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-y", "-cpuflags", "0", "-f", "lavfi", "-i", spec, "-frames:v", "2", "-c:v", "mjpeg", avi)
	require.Equal(t, 0, code, stderr)
	got := systemFF(t, "-i", avi, "-frames:v", "2", "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	require.Len(t, got, 16*16*3*2)
	code, _, stderr = runFF(t, "-hide_banner", "-loglevel", "error", "-y", "-cpuflags", "0", "-f", "lavfi", "-i", spec, "-frames:v", "2", "-c:v", "mjpeg", mov)
	require.Equal(t, 0, code, stderr)
	got = systemFF(t, "-i", mov, "-frames:v", "2", "-f", "rawvideo", "-pix_fmt", "rgb24", "-")
	require.Len(t, got, 16*16*3*2)
}

func TestG722CLIMatchesFFmpeg(t *testing.T) {
	spec := "sine=frequency=440:sample_rate=16000:duration=0.04"
	want := systemFF(t, "-f", "lavfi", "-i", spec, "-c:a", "g722", "-f", "g722", "-")
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0",
		"-f", "lavfi", "-i", spec, "-c:a", "g722", "-f", "g722", "-")
	require.Equal(t, 0, code, stderr)
	require.Equal(t, want, []byte(stdout))
	path := filepath.Join(t.TempDir(), "a.g722")
	require.NoError(t, os.WriteFile(path, want, 0o644))
	decoded := systemFF(t, "-f", "g722", "-i", path, "-f", "s16le", "-")
	code, stdout, stderr = runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0",
		"-f", "g722", "-i", path, "-f", "s16le", "-")
	require.Equal(t, 0, code, stderr)
	require.Equal(t, decoded, []byte(stdout))
}

func TestIMAQTCLIMatchesFFmpeg(t *testing.T) {
	const frame = 64 * 3
	pcm := systemFF(t, "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1", "-f", "s16le", "-")
	pcm = pcm[:frame*2]
	dir := t.TempDir()
	path := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(path, pcm, 0o644))
	wantPath := filepath.Join(dir, "want.aiff")
	gotPath := filepath.Join(dir, "got.aiff")
	args := []string{"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", path, "-c:a", "adpcm_ima_qt", "-f", "aiff"}
	_ = systemFF(t, append(args, wantPath)...)
	code, _, stderr := runFF(t, append(append([]string{"-hide_banner", "-loglevel", "error", "-cpuflags", "0", "-y"}, args...), gotPath)...)
	require.Equal(t, 0, code, stderr)
	want, err := os.ReadFile(wantPath)
	require.NoError(t, err)
	got, err := os.ReadFile(gotPath)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestYamahaCLIMatchesFFmpeg(t *testing.T) {
	const frame = 1024 * 2
	pcm := systemFF(t, "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1", "-f", "s16le", "-")
	pcm = pcm[:frame*2]
	path := filepath.Join(t.TempDir(), "a.s16")
	require.NoError(t, os.WriteFile(path, pcm, 0o644))
	args := []string{"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", path, "-c:a", "adpcm_yamaha", "-f", "wav", "-"}
	want := systemFF(t, args...)
	code, stdout, stderr := runFF(t, append([]string{"-hide_banner", "-loglevel", "error", "-cpuflags", "0"}, args...)...)
	require.Equal(t, 0, code, stderr)
	require.Equal(t, wavData(t, want), wavData(t, []byte(stdout)))
}

func TestMSADPCMCLIMatchesFFmpeg(t *testing.T) {
	const frame = (1024-7)*2 + 2
	pcm := systemFF(t, "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1", "-f", "s16le", "-")
	pcm = pcm[:frame*2]
	path := filepath.Join(t.TempDir(), "a.s16")
	require.NoError(t, os.WriteFile(path, pcm, 0o644))
	args := []string{"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", path, "-c:a", "adpcm_ms", "-f", "wav", "-"}
	want := systemFF(t, args...)
	code, stdout, stderr := runFF(t, append([]string{"-hide_banner", "-loglevel", "error", "-cpuflags", "0"}, args...)...)
	require.Equal(t, 0, code, stderr)
	require.Equal(t, wavData(t, want), wavData(t, []byte(stdout)))
}

func TestIMAWAVCLIMatchesFFmpeg(t *testing.T) {
	const frame = (1024-4)*2 + 1
	pcm := systemFF(t, "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1", "-f", "s16le", "-")
	pcm = pcm[:frame*2]
	path := filepath.Join(t.TempDir(), "a.s16")
	require.NoError(t, os.WriteFile(path, pcm, 0o644))
	args := []string{"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", path, "-c:a", "adpcm_ima_wav", "-f", "wav", "-"}
	want := systemFF(t, args...)
	code, stdout, stderr := runFF(t, append([]string{"-hide_banner", "-loglevel", "error", "-cpuflags", "0"}, args...)...)
	require.Equal(t, 0, code, stderr)
	require.Equal(t, wavData(t, want), wavData(t, []byte(stdout)))
}

func TestMP2FixedCLIMatchesFFmpeg(t *testing.T) {
	spec := "sine=frequency=440:sample_rate=44100:duration=0.05"
	want := systemFF(t, "-f", "lavfi", "-i", spec, "-c:a", "mp2fixed", "-f", "mp2", "-")
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0",
		"-f", "lavfi", "-i", spec, "-c:a", "mp2fixed", "-f", "mp2", "-")
	require.Equal(t, 0, code, stderr)
	require.Equal(t, want, []byte(stdout))
}

func TestGSMMSEncodeMatchesFFmpeg(t *testing.T) {
	spec := "sine=frequency=440:sample_rate=8000:duration=0.08"
	wantWav := systemFF(t, "-f", "lavfi", "-i", spec, "-c:a", "libgsm_ms", "-f", "wav", "-")
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0",
		"-f", "lavfi", "-i", spec, "-c:a", "libgsm_ms", "-f", "wav", "-")
	require.Equal(t, 0, code, stderr)
	require.Equal(t, wavData(t, wantWav), wavData(t, []byte(stdout)))
}

func wavData(t *testing.T, b []byte) []byte {
	t.Helper()
	off := 12
	for off+8 <= len(b) {
		name := string(b[off : off+4])
		n := int(binary.LittleEndian.Uint32(b[off+4:]))
		off += 8
		if n == int(^uint32(0)) {
			n = len(b) - off
		}
		if name == "data" {
			return b[off : off+n]
		}
		off += n
	}
	t.Fatal("wav data missing")
	return nil
}

func TestGSMEncodeMatchesFFmpeg(t *testing.T) {
	spec := "sine=frequency=440:sample_rate=8000:duration=0.04"
	want := systemFF(t, "-f", "lavfi", "-i", spec, "-c:a", "libgsm", "-f", "gsm", "-")
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0",
		"-f", "lavfi", "-i", spec, "-c:a", "libgsm", "-f", "gsm", "-")
	require.Equal(t, 0, code, stderr)
	require.Equal(t, want, []byte(stdout))
}

func TestGSMMatchesFFmpeg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.gsm")
	spec := "sine=frequency=440:sample_rate=8000:duration=0.08"
	_ = systemFF(t, "-f", "lavfi", "-i", spec, "-c:a", "libgsm", "-f", "gsm", path)
	want := systemFF(t, "-f", "gsm", "-i", path, "-f", "s16le", "-")
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-f", "gsm", "-i", path, "-f", "s16le", "-")
	require.Equal(t, 0, code, stderr)
	require.Equal(t, want, []byte(stdout))
}

func TestHEVCRoundTrip(t *testing.T) {
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0",
		"-f", "lavfi", "-i", "color=c=red:s=16x16:r=25:d=0.04", "-frames:v", "1",
		"-c:v", "libx265", "-f", "hevc", "-")
	require.Equal(t, 0, code, stderr)
	path := filepath.Join(t.TempDir(), "a.hevc")
	require.NoError(t, os.WriteFile(path, []byte(stdout), 0o644))
	got := systemFF(t, "-i", path, "-frames:v", "1", "-f", "rawvideo", "-pix_fmt", "yuv420p", "-")
	require.Len(t, got, 16*16+8*8+8*8)
}

func TestH264RoundTrip(t *testing.T) {
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0",
		"-f", "lavfi", "-i", "color=c=red:s=16x16:r=25:d=0.04", "-frames:v", "1",
		"-c:v", "libx264", "-f", "h264", "-")
	require.Equal(t, 0, code, stderr)
	path := filepath.Join(t.TempDir(), "a.h264")
	require.NoError(t, os.WriteFile(path, []byte(stdout), 0o644))
	got := systemFF(t, "-i", path, "-frames:v", "1", "-f", "rawvideo", "-pix_fmt", "yuv420p", "-")
	require.Len(t, got, 16*16+8*8+8*8)
}

func TestMapAudio(t *testing.T) {
	code, stdout, stderr := runFF(t, "-hide_banner", "-loglevel", "error", "-cpuflags", "0",
		"-f", "lavfi", "-i", "testsrc=size=16x16:rate=25:duration=0.04",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=0.05",
		"-map", "1:a", "-f", "s16le", "-")
	require.Equal(t, 0, code, stderr)
	want := systemFF(t, "-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=0.05", "-f", "s16le", "-")
	require.Equal(t, want, []byte(stdout))
}
