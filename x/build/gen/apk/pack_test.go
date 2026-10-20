package apk

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/stretchr/testify/require"
)

func TestAssembleGoAPK(t *testing.T) {
	work := t.TempDir()
	libDir := filepath.Join(work, "app", "src", "main", "jniLibs", "arm64-v8a")
	require.NoError(t, os.MkdirAll(libDir, 0o755))
	so := []byte{0x7f, 'E', 'L', 'F', 0, 1, 2, 3}
	require.NoError(t, os.WriteFile(filepath.Join(libDir, "libeletrocromo.so"), so, 0o644))
	res := filepath.Join(work, "app", "src", "main", "res")
	for _, dir := range []string{"mipmap-mdpi", "mipmap-hdpi", "mipmap-xhdpi", "mipmap-xxhdpi", "mipmap-xxxhdpi"} {
		require.NoError(t, os.MkdirAll(filepath.Join(res, dir), 0o755))
		writePNG(t, filepath.Join(res, dir, "ic_launcher.png"))
	}
	out := filepath.Join(t.TempDir(), "demo.apk")
	cfg := Config{
		PackageID:    "br.tec.lew.demo",
		AppName:      "Demo",
		VersionName:  "1.2.3",
		VersionCode:  5,
		Capabilities: common.Capabilities{},
	}
	require.NoError(t, assembleGoAPK(cfg, work, out))

	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, "PK", string(raw[:2]))
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	require.NoError(t, err)
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	for _, name := range []string{
		"AndroidManifest.xml",
		"classes.dex",
		"resources.arsc",
		"res/layout/activity_main.xml",
		"res/xml/file_paths.xml",
		"res/xml/network_security_config.xml",
		"res/mipmap-mdpi/ic_launcher.png",
		"res/mipmap-xxxhdpi/ic_launcher.png",
		"lib/arm64-v8a/libeletrocromo.so",
	} {
		_, ok := files[name]
		require.True(t, ok, "missing %s", name)
	}
	arsc := files["resources.arsc"]
	require.Equal(t, zip.Store, arsc.Method)
	off, err := arsc.DataOffset()
	require.NoError(t, err)
	require.Zero(t, off%4)
	man := readZip(t, files["AndroidManifest.xml"])
	pool := xmlPoolStrings(t, man)
	require.Contains(t, pool, "br.tec.lew.demo")
	require.Contains(t, pool, "1.2.3")
	require.Contains(t, pool, "lewkit.MainActivity")
	require.Contains(t, pool, "lewkit.FileProvider")
	require.Contains(t, pool, "br.tec.lew.demo.fileprovider")
	root := xmlRoot(t, man, pool)
	require.Equal(t, "manifest", root.name)
	require.Equal(t, "br.tec.lew.demo", root.attr["package"])
	iconRaw, iconData := manifestAttr(t, man, "application", "icon")
	require.Equal(t, uint32(xmlNoIndex), iconRaw)
	require.Equal(t, resIDIcon, iconData)
	labelRaw, labelData := manifestAttr(t, man, "application", "label")
	require.Equal(t, uint32(xmlNoIndex), labelRaw)
	require.Equal(t, resIDAppName, labelData)
	attrIDs := manifestAndroidAttrIDs(t, man, "application")
	require.Contains(t, attrIDs, attrIcon)
	for i := 1; i < len(attrIDs); i++ {
		require.LessOrEqual(t, attrIDs[i-1], attrIDs[i])
	}
	table := readZip(t, arsc)
	require.Equal(t, "br.tec.lew.demo", arscPackageName(t, table))
	tablePool := arscPoolStrings(t, table)
	require.Contains(t, tablePool, "Demo")
	require.Contains(t, tablePool, "Starting…")
	require.Contains(t, tablePool, "res/layout/activity_main.xml")
	require.Contains(t, string(hostDex), "lewkit/MainActivity")
	require.Contains(t, string(hostDex), "lewkit/Hook")
	require.Contains(t, string(hostDex), "lewkit/Host")
	require.Contains(t, string(hostDex), "androidx/swiperefreshlayout/widget/SwipeRefreshLayout")

	id, err := sign.Generate("Demo")
	require.NoError(t, err)
	signed, err := id.SignAPK(raw)
	require.NoError(t, err)
	require.NoError(t, id.VerifyAPK(signed))
}

func TestBuildAssemblesGoApp(t *testing.T) {
	mainDir, err := filepath.Abs("testdata/nocgo")
	require.NoError(t, err)
	out := filepath.Join(t.TempDir(), "Demo.apk")
	result, err := Build(t.Context(), BuildOptions{
		Config: Config{
			PackageID:   "br.tec.lew.demo",
			AppName:     "Demo",
			VersionName: "1.2.3",
			VersionCode: 5,
			GoMain:      mainDir,
		},
		BaseDir: mainDir,
		WorkDir: t.TempDir(),
		OutAPK:  out,
	})
	require.NoError(t, err)
	require.Equal(t, out, result.APKPath)
	zr, err := zip.OpenReader(out)
	require.NoError(t, err)
	defer zr.Close()
	var so []byte
	for _, f := range zr.File {
		if f.Name == "lib/arm64-v8a/libeletrocromo.so" {
			rc, err := f.Open()
			require.NoError(t, err)
			so, err = io.ReadAll(rc)
			rc.Close()
			require.NoError(t, err)
		}
	}
	require.Greater(t, len(so), 4)
	require.Equal(t, []byte{0x7f, 'E', 'L', 'F'}, so[:4])
}

func writePNG(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 2, 2))))
	require.NoError(t, f.Close())
}

func readZip(t *testing.T, f *zip.File) []byte {
	t.Helper()
	rc, err := f.Open()
	require.NoError(t, err)
	defer rc.Close()
	raw, err := io.ReadAll(rc)
	require.NoError(t, err)
	return raw
}

type xmlEl struct {
	name string
	attr map[string]string
	kids []xmlEl
}

func xmlPoolStrings(t *testing.T, raw []byte) []string {
	t.Helper()
	require.Greater(t, len(raw), 8)
	require.Equal(t, uint16(xmlChunkXML), binary.LittleEndian.Uint16(raw))
	strs, _, err := readStringPool(raw, 8)
	require.NoError(t, err)
	return strs
}

func manifestAndroidAttrIDs(t *testing.T, raw []byte, element string) []uint32 {
	t.Helper()
	pool, off, err := readStringPool(raw, 8)
	require.NoError(t, err)
	require.Equal(t, uint16(xmlChunkResMap), binary.LittleEndian.Uint16(raw[off:]))
	mapSize := int(binary.LittleEndian.Uint32(raw[off+4:]))
	ids := make([]uint32, (mapSize-8)/4)
	for i := range ids {
		ids[i] = binary.LittleEndian.Uint32(raw[off+8+4*i:])
	}
	off += mapSize
	for off+8 <= len(raw) {
		kind := binary.LittleEndian.Uint16(raw[off:])
		size := int(binary.LittleEndian.Uint32(raw[off+4:]))
		require.GreaterOrEqual(t, size, 8)
		if kind == xmlStartElement && pool[binary.LittleEndian.Uint32(raw[off+20:])] == element {
			count := int(binary.LittleEndian.Uint16(raw[off+28:]))
			start := int(binary.LittleEndian.Uint16(raw[off+24:]))
			base := off + 16 + start
			out := make([]uint32, 0, count)
			for i := range count {
				at := base + 20*i
				nameIdx := binary.LittleEndian.Uint32(raw[at+4:])
				require.Less(t, int(nameIdx), len(ids))
				out = append(out, ids[nameIdx])
			}
			return out
		}
		off += size
	}
	require.FailNow(t, "manifest element "+element)
	return nil
}

func manifestAttr(t *testing.T, raw []byte, element, name string) (rawIdx, data uint32) {
	t.Helper()
	pool, off, err := readStringPool(raw, 8)
	require.NoError(t, err)
	if off+8 <= len(raw) && binary.LittleEndian.Uint16(raw[off:]) == xmlChunkResMap {
		off += int(binary.LittleEndian.Uint32(raw[off+4:]))
	}
	for off+8 <= len(raw) {
		kind := binary.LittleEndian.Uint16(raw[off:])
		size := int(binary.LittleEndian.Uint32(raw[off+4:]))
		require.GreaterOrEqual(t, size, 8)
		if kind == xmlStartElement && pool[binary.LittleEndian.Uint32(raw[off+20:])] == element {
			count := int(binary.LittleEndian.Uint16(raw[off+28:]))
			start := int(binary.LittleEndian.Uint16(raw[off+24:]))
			base := off + 16 + start
			for i := range count {
				at := base + 20*i
				if pool[binary.LittleEndian.Uint32(raw[at+4:])] != name {
					continue
				}
				return binary.LittleEndian.Uint32(raw[at+8:]), binary.LittleEndian.Uint32(raw[at+16:])
			}
		}
		off += size
	}
	require.FailNow(t, "manifest attribute "+element+" "+name)
	return 0, 0
}

func xmlRoot(t *testing.T, raw []byte, pool []string) xmlEl {
	t.Helper()
	off := 8
	strs, next, err := readStringPool(raw, off)
	require.NoError(t, err)
	_ = strs
	off = next
	if off+8 <= len(raw) && binary.LittleEndian.Uint16(raw[off:]) == xmlChunkResMap {
		off += int(binary.LittleEndian.Uint32(raw[off+4:]))
	}
	var stack []*xmlEl
	var root xmlEl
	for off+8 <= len(raw) && off < len(raw) {
		kind := binary.LittleEndian.Uint16(raw[off:])
		size := int(binary.LittleEndian.Uint32(raw[off+4:]))
		require.GreaterOrEqual(t, size, 8)
		require.LessOrEqual(t, off+size, len(raw))
		switch kind {
		case xmlStartElement:
			name := pool[binary.LittleEndian.Uint32(raw[off+20:])]
			count := int(binary.LittleEndian.Uint16(raw[off+28:]))
			el := xmlEl{name: name, attr: map[string]string{}}
			base := off + 36
			for i := 0; i < count; i++ {
				at := base + 20*i
				ns := binary.LittleEndian.Uint32(raw[at:])
				nidx := binary.LittleEndian.Uint32(raw[at+4:])
				rawIdx := binary.LittleEndian.Uint32(raw[at+8:])
				key := pool[nidx]
				if ns != xmlNoIndex {
					key = "android:" + key
				}
				if rawIdx != xmlNoIndex {
					el.attr[key] = pool[rawIdx]
				}
			}
			if len(stack) == 0 {
				root = el
				stack = append(stack, &root)
			} else {
				parent := stack[len(stack)-1]
				parent.kids = append(parent.kids, el)
				stack = append(stack, &parent.kids[len(parent.kids)-1])
			}
		case xmlEndElement:
			stack = stack[:len(stack)-1]
		}
		off += size
		if kind == xmlChunkXML {
			break
		}
	}
	return root
}

func arscPoolStrings(t *testing.T, raw []byte) []string {
	t.Helper()
	require.Equal(t, uint16(chunkTable), binary.LittleEndian.Uint16(raw))
	strs, _, err := readStringPool(raw, 12)
	require.NoError(t, err)
	return strs
}

func arscPackageName(t *testing.T, raw []byte) string {
	t.Helper()
	_, off, err := readStringPool(raw, 12)
	require.NoError(t, err)
	require.Greater(t, len(raw), off+268)
	var b strings.Builder
	for i := range 128 {
		c := binary.LittleEndian.Uint16(raw[off+12+2*i:])
		if c == 0 {
			break
		}
		b.WriteRune(rune(c))
	}
	return b.String()
}

func readStringPool(raw []byte, off int) ([]string, int, error) {
	if off+28 > len(raw) {
		return nil, 0, errShort
	}
	if binary.LittleEndian.Uint16(raw[off:]) != xmlChunkString {
		return nil, 0, errShort
	}
	size := int(binary.LittleEndian.Uint32(raw[off+4:]))
	count := int(binary.LittleEndian.Uint32(raw[off+8:]))
	start := int(binary.LittleEndian.Uint32(raw[off+20:]))
	out := make([]string, count)
	for i := 0; i < count; i++ {
		rel := int(binary.LittleEndian.Uint32(raw[off+28+4*i:]))
		out[i] = decodeUTF16(raw[off+start+rel:])
	}
	return out, off + size, nil
}

func decodeUTF16(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	n := int(binary.LittleEndian.Uint16(b))
	b = b[2:]
	if n&0x8000 != 0 {
		if len(b) < 2 {
			return ""
		}
		n = ((n & 0x7fff) << 16) | int(binary.LittleEndian.Uint16(b))
		b = b[2:]
	}
	if len(b) < n*2 {
		return ""
	}
	u := make([]uint16, n)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(u))
}

var errShort = errAssembleShort()

func errAssembleShort() error { return ErrAssembleAPK }
