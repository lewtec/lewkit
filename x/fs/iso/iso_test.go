package iso

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestWriteRoundTrip(t *testing.T) {
	t.Parallel()
	payload := append([]byte("MZ\x00\n"), bytes.Repeat([]byte{0x11}, 1000)...)
	image, err := Write(fstest.MapFS{
		"EFI/BOOT/BOOTX64.EFI": &fstest.MapFile{Data: payload},
		"ZZZ.TXT":              &fstest.MapFile{Data: []byte("tail")},
	})
	require.NoError(t, err)
	require.Equal(t, "CD001", string(image[0x8001:0x8006]))
	require.Contains(t, string(image[17*2048:17*2048+64]), "EL TORITO SPECIFICATION")

	catLBA := binary.LittleEndian.Uint32(image[17*2048+71:])
	cat := image[catLBA*2048:]
	require.Equal(t, byte(0x01), cat[0])
	require.Equal(t, byte(0xEF), cat[1])
	require.Equal(t, byte(0x55), cat[30])
	require.Equal(t, byte(0xAA), cat[31])
	var sum uint32
	for i := 0; i < 32; i += 2 {
		sum += uint32(binary.LittleEndian.Uint16(cat[i:]))
	}
	require.Zero(t, sum&0xFFFF)
	require.Equal(t, byte(0x88), cat[32])
	require.Equal(t, byte(0x00), cat[33])

	got := isoFile(t, image, "EFI", "BOOT", "BOOTX64.EFI;1")
	require.Equal(t, payload, got)
	require.Equal(t, []byte("tail"), isoFile(t, image, "ZZZ.TXT;1"))
	fat := isoFile(t, image, "EFIBOOT.IMG;1")
	require.Equal(t, payload, fatFile(t, fat, "EFI", "BOOT", "BOOTX64.EFI"))
	require.Equal(t, []byte("tail"), fatFile(t, fat, "ZZZ.TXT"))
	require.Equal(t, elToritoSectors(len(fat)), binary.LittleEndian.Uint16(cat[38:]))

	read, err := ReadFile(image, "efi/boot/bootx64.efi")
	require.NoError(t, err)
	require.Equal(t, payload, read)
}

func TestWriteSubdirCluster(t *testing.T) {
	t.Parallel()
	tree := fstest.MapFS{}
	for i := 0; i < 20; i++ {
		tree[fmt.Sprintf("SUB/F%02d.BIN", i)] = &fstest.MapFile{Data: []byte{byte(i)}}
	}
	image, err := Write(tree)
	require.NoError(t, err)
	got, err := ReadFile(image, "SUB/F19.BIN")
	require.NoError(t, err)
	require.Equal(t, []byte{19}, got)
	fat := isoFile(t, image, "EFIBOOT.IMG;1")
	require.Equal(t, []byte{19}, fatFile(t, fat, "SUB", "F19.BIN"))
}

func TestWriteRejects(t *testing.T) {
	t.Parallel()
	_, err := Write(nil)
	require.Error(t, err)
	_, err = Write(fstest.MapFS{})
	require.Error(t, err)

	_, err = Write(fstest.MapFS{
		"LINK.TXT": &fstest.MapFile{Data: []byte("A.TXT"), Mode: fs.ModeSymlink},
	})
	require.Error(t, err)

	_, err = Write(fstest.MapFS{
		"EFIBOOT.IMG": &fstest.MapFile{Data: []byte("nope")},
	})
	require.Error(t, err)
}

func TestWriteLongName(t *testing.T) {
	t.Parallel()
	image, err := Write(fstest.MapFS{
		"A.B.C": &fstest.MapFile{Data: []byte("x")},
	})
	require.NoError(t, err)
	got, err := ReadFile(image, "A.B.C")
	require.NoError(t, err)
	require.Equal(t, []byte("x"), got)
}

func TestWriteWideDirectory(t *testing.T) {
	t.Parallel()
	tree := fstest.MapFS{}
	for i := 0; i < 47; i++ {
		tree[fmt.Sprintf("F%02d.TXT", i)] = &fstest.MapFile{Data: []byte{byte(i)}}
	}
	image, err := Write(tree)
	require.NoError(t, err)
	got, err := ReadFile(image, "F46.TXT")
	require.NoError(t, err)
	require.Equal(t, []byte{46}, got)
}

func elToritoSectors(n int) uint16 {
	sectors := (n + 511) / 512
	if sectors < 2 || sectors > 65535 {
		return 0
	}
	return uint16(sectors)
}

func isoFile(t *testing.T, image []byte, parts ...string) []byte {
	t.Helper()
	rec := image[16*2048+156:]
	for _, part := range parts {
		extent := binary.LittleEndian.Uint32(rec[2:])
		size := binary.LittleEndian.Uint32(rec[10:])
		dir := image[int(extent)*2048 : int(extent)*2048+int(size)]
		next, ok := findISO(dir, part)
		require.True(t, ok, part)
		rec = next
	}
	extent := binary.LittleEndian.Uint32(rec[2:])
	size := binary.LittleEndian.Uint32(rec[10:])
	off := int(extent) * 2048
	return append([]byte{}, image[off:off+int(size)]...)
}

func findISO(dir []byte, name string) ([]byte, bool) {
	for off := 0; off+33 <= len(dir); {
		n := int(dir[off])
		if n == 0 {
			return nil, false
		}
		nameLen := int(dir[off+32])
		if string(dir[off+33:off+33+nameLen]) == name {
			return dir[off : off+n], true
		}
		off += n
	}
	return nil, false
}

func fatFile(t *testing.T, img []byte, parts ...string) []byte {
	t.Helper()
	bps := int(binary.LittleEndian.Uint16(img[11:]))
	spc := int(img[13])
	reserved := int(binary.LittleEndian.Uint16(img[14:]))
	nfats := int(img[16])
	rootEnt := int(binary.LittleEndian.Uint16(img[17:]))
	spf := int(binary.LittleEndian.Uint16(img[22:]))
	rootOff := (reserved + nfats*spf) * bps
	dataOff := rootOff + rootEnt*32
	clusterSize := spc * bps
	dir := img[rootOff : rootOff+rootEnt*32]
	var start int
	var size int
	for i, part := range parts {
		name := fatName(part)
		ent, ok := findFAT(dir, name)
		require.True(t, ok, part)
		start = int(binary.LittleEndian.Uint16(ent[26:]))
		size = int(binary.LittleEndian.Uint32(ent[28:]))
		if i == len(parts)-1 {
			break
		}
		require.GreaterOrEqual(t, start, 2, part)
		got := binary.LittleEndian.Uint16(img[reserved*bps+start*2:])
		require.Equal(t, uint16(0xFFFF), got, part)
		dir = fatCluster(img, start, dataOff, clusterSize)
	}
	var out []byte
	c := start
	for c >= 2 && c < 0xFFF8 {
		out = append(out, fatCluster(img, c, dataOff, clusterSize)...)
		c = int(binary.LittleEndian.Uint16(img[reserved*bps+c*2:]))
	}
	require.GreaterOrEqual(t, len(out), size)
	return out[:size]
}

func fatName(name string) string {
	base, ext, ok := strings.Cut(name, ".")
	if !ok {
		return fmt.Sprintf("%-11s", name)
	}
	return fmt.Sprintf("%-8s%-3s", base, ext)
}

func findFAT(dir []byte, name string) ([]byte, bool) {
	for off := 0; off+32 <= len(dir); off += 32 {
		if dir[off] == 0 {
			return nil, false
		}
		if string(dir[off:off+11]) == name {
			return dir[off : off+32], true
		}
	}
	return nil, false
}

func fatCluster(img []byte, cluster, dataOff, clusterSize int) []byte {
	off := dataOff + (cluster-2)*clusterSize
	return img[off : off+clusterSize]
}
