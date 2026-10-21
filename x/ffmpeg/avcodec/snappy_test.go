package avcodec

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestSnappyRoundTrip(t *testing.T) {
	samples := [][]byte{
		[]byte("a"),
		[]byte("hello snappy"),
		bytes.Repeat([]byte{0, 1, 2, 3, 4, 5, 6, 7}, 40),
		bytes.Repeat([]byte("abcd"), 200),
	}
	for _, src := range samples {
		enc := SnappyCompress(src)
		avutil.SetScalar(true)
		got, err := SnappyUncompress(enc)
		require.NoError(t, err)
		require.Equal(t, src, got)
		avutil.SetScalar(false)
		got, err = SnappyUncompress(enc)
		require.NoError(t, err)
		require.Equal(t, src, got)
	}
}

func TestSnappyMatchesLibsnappy(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "snappy-oracle")
	src := []byte(`#include <stdio.h>
#include <stdlib.h>
#include <stddef.h>
typedef enum { SNAPPY_OK = 0, SNAPPY_INVALID_INPUT = 1, SNAPPY_BUFFER_TOO_SMALL = 2 } snappy_status;
snappy_status snappy_compress(const char *input, size_t input_length, char *compressed, size_t *compressed_length);
snappy_status snappy_uncompress(const char *compressed, size_t compressed_length, char *uncompressed, size_t *uncompressed_length);
size_t snappy_max_compressed_length(size_t source_length);
snappy_status snappy_uncompressed_length(const char *compressed, size_t compressed_length, size_t *result);
int main(int argc, char **argv) {
    if (argc < 2) return 2;
    FILE *in = fopen(argv[1], "rb");
    if (!in) return 2;
    fseek(in, 0, SEEK_END);
    long n = ftell(in);
    fseek(in, 0, SEEK_SET);
    char *raw = malloc(n);
    if (fread(raw, 1, n, in) != (size_t)n) return 2;
    fclose(in);
    if (argv[2][0] == 'c') {
        size_t cap = snappy_max_compressed_length(n);
        char *out = malloc(cap);
        if (snappy_compress(raw, n, out, &cap) != SNAPPY_OK) return 3;
        fwrite(out, 1, cap, stdout);
        return 0;
    }
    size_t cap = 0;
    if (snappy_uncompressed_length(raw, n, &cap) != SNAPPY_OK) return 3;
    char *out = malloc(cap);
    if (snappy_uncompress(raw, n, out, &cap) != SNAPPY_OK) return 3;
    fwrite(out, 1, cap, stdout);
    return 0;
}
`)
	cfile := filepath.Join(t.TempDir(), "o.c")
	require.NoError(t, os.WriteFile(cfile, src, 0o644))
	cmd := exec.Command("gcc", "-O2", "-o", bin, cfile, "/usr/lib/x86_64-linux-gnu/libsnappy.so.1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		test.Skipf(t, "libsnappy headers: %s", out)
	}
	plain := bytes.Repeat([]byte("lewkit-snappy-block:"), 80)
	plain = append(plain, []byte{0, 0, 1, 0xff, 0x7f}...)
	path := filepath.Join(t.TempDir(), "in")
	require.NoError(t, os.WriteFile(path, plain, 0o644))

	comp, err := exec.Command(bin, path, "c").Output()
	require.NoError(t, err)
	got, err := SnappyUncompress(comp)
	require.NoError(t, err)
	require.Equal(t, plain, got)

	ours := SnappyCompress(plain)
	require.NoError(t, os.WriteFile(path, ours, 0o644))
	back, err := exec.Command(bin, path, "u").Output()
	require.NoError(t, err)
	require.Equal(t, plain, back)
}
