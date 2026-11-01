package uki

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseAPKINDEX(t *testing.T) {
	sum := bytes.Repeat([]byte{1}, sha1.Size)
	field := "Q1" + base64.StdEncoding.EncodeToString(sum)
	text := "C:" + field + "\nP:musl\nV:1.2.6-r3\nS:10\nA:x86_64\n\n" +
		"P:skip\n\n" +
		"C:" + field + "\nP:musl\nV:9.9.9-r9\nS:10\n"
	index, err := parseAPKINDEX(text)
	require.NoError(t, err)
	require.Equal(t, "1.2.6-r3", index["musl"].version)
	require.Equal(t, int64(10), index["musl"].size)
	require.Equal(t, sum, index["musl"].checksum)
	require.NotContains(t, index, "skip")

	_, err = parseAPKINDEX("C:nope\nP:musl\nV:1\nS:1\n")
	require.ErrorIs(t, err, errUKI)
	_, err = parseAPKINDEX(" \n")
	require.ErrorIs(t, err, errUKI)
}

func TestAlpineApkChecksum(t *testing.T) {
	data := gzipMember(t, []byte("kernel-bytes"))
	dataSum := sha256.Sum256(data)
	control := gzipMember(t, pkginfoTar(t, hex.EncodeToString(dataSum[:])))
	raw := append(append(gzipMember(t, []byte("signature")), control...), data...)
	controlSum := sha1.Sum(control)
	pkg := alpinePackage{
		name:     "linux-virt",
		version:  "1-r0",
		repo:     "main",
		size:     int64(len(raw)),
		checksum: controlSum[:],
	}
	path := filepath.Join(t.TempDir(), "linux-virt.apk")
	require.NoError(t, os.WriteFile(path, raw, 0o644))
	require.NoError(t, verifyAlpineApk(path, pkg))

	raw[len(raw)-1] ^= 0xff
	require.NoError(t, os.WriteFile(path, raw, 0o644))
	require.ErrorIs(t, verifyAlpineApk(path, pkg), errUKI)
}

func gzipMember(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	require.NoError(t, err)
	_, err = zw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func pkginfoTar(t *testing.T, datahash string) []byte {
	t.Helper()
	body := []byte("datahash = " + datahash + "\n")
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: ".PKGINFO",
		Mode: 0o644,
		Size: int64(len(body)),
	}))
	_, err := tw.Write(body)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	return buf.Bytes()
}
