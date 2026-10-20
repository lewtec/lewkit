package apk

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHostDexResolvesClasses(t *testing.T) {
	missing := unresolvedDexClasses(t, hostDex)
	require.Empty(t, missing)
}

func TestHostActivityHidesTitle(t *testing.T) {
	require.True(t, dexHasMethod(t, hostDex, "Llewkit/HostActivity;", "onCreate", []string{"Landroid/os/Bundle;"}, "V"))
}

func unresolvedDexClasses(t *testing.T, raw []byte) []string {
	t.Helper()
	require.Greater(t, len(raw), 112)
	require.Equal(t, "dex\n", string(raw[:4]))
	stringIDs := int(binary.LittleEndian.Uint32(raw[56:]))
	stringOff := int(binary.LittleEndian.Uint32(raw[60:]))
	typeIDs := int(binary.LittleEndian.Uint32(raw[64:]))
	typeOff := int(binary.LittleEndian.Uint32(raw[68:]))
	classDefs := int(binary.LittleEndian.Uint32(raw[96:]))
	classOff := int(binary.LittleEndian.Uint32(raw[100:]))

	stringsAt := make([]string, stringIDs)
	for i := range stringsAt {
		off := int(binary.LittleEndian.Uint32(raw[stringOff+4*i:]))
		_, off = dexULEB(t, raw, off)
		end := off
		for end < len(raw) && raw[end] != 0 {
			end++
		}
		stringsAt[i] = string(raw[off:end])
	}
	typeName := func(i uint32) string {
		if i == 0xffffffff {
			return ""
		}
		require.Less(t, int(i), typeIDs)
		idx := binary.LittleEndian.Uint32(raw[typeOff+4*int(i):])
		require.Less(t, int(idx), len(stringsAt))
		return stringsAt[idx]
	}
	defined := map[string]bool{}
	for i := range classDefs {
		classIdx := binary.LittleEndian.Uint32(raw[classOff+32*i:])
		for _, name := range dexInternals(typeName(classIdx)) {
			defined[name] = true
		}
	}
	var missing []string
	seen := map[string]bool{}
	for i := range typeIDs {
		desc := typeName(uint32(i))
		for _, name := range dexInternals(desc) {
			if defined[name] || dexFramework(name) || seen[name] {
				continue
			}
			seen[name] = true
			missing = append(missing, name)
		}
	}
	return missing
}

func dexULEB(t *testing.T, raw []byte, off int) (int, int) {
	t.Helper()
	n := 0
	shift := 0
	for {
		require.Less(t, off, len(raw))
		b := raw[off]
		off++
		n |= int(b&0x7f) << shift
		if b < 0x80 {
			return n, off
		}
		shift += 7
	}
}

func dexInternals(desc string) []string {
	for strings.HasPrefix(desc, "[") {
		desc = desc[1:]
	}
	if strings.HasPrefix(desc, "L") && strings.HasSuffix(desc, ";") {
		return []string{desc[1 : len(desc)-1]}
	}
	return nil
}

func dexFramework(name string) bool {
	if strings.HasPrefix(name, "java/") || strings.HasPrefix(name, "javax/") || strings.HasPrefix(name, "dalvik/") || strings.HasPrefix(name, "sun/") || strings.HasPrefix(name, "libcore/") || strings.HasPrefix(name, "jdk/") {
		return true
	}
	if strings.HasPrefix(name, "android/") && !strings.HasPrefix(name, "androidx/") {
		return true
	}
	if strings.HasPrefix(name, "org/xml") || strings.HasPrefix(name, "org/json") || strings.HasPrefix(name, "org/w3c") {
		return true
	}
	return false
}

func dexHasMethod(t *testing.T, raw []byte, classDesc, name string, params []string, returnDesc string) bool {
	t.Helper()
	stringsAt := dexStrings(t, raw)
	typeOff := int(binary.LittleEndian.Uint32(raw[68:]))
	typeName := func(i uint32) string {
		idx := binary.LittleEndian.Uint32(raw[typeOff+4*int(i):])
		return stringsAt[idx]
	}
	protoOff := int(binary.LittleEndian.Uint32(raw[76:]))
	protoMatch := func(idx int) bool {
		base := protoOff + 12*idx
		ret := typeName(binary.LittleEndian.Uint32(raw[base+4:]))
		if ret != returnDesc {
			return false
		}
		paramsOff := int(binary.LittleEndian.Uint32(raw[base+8:]))
		if paramsOff == 0 {
			return len(params) == 0
		}
		n := int(binary.LittleEndian.Uint32(raw[paramsOff:]))
		if n != len(params) {
			return false
		}
		for i := range params {
			got := typeName(uint32(binary.LittleEndian.Uint16(raw[paramsOff+4+2*i:])))
			if got != params[i] {
				return false
			}
		}
		return true
	}
	methodSize := int(binary.LittleEndian.Uint32(raw[88:]))
	methodOff := int(binary.LittleEndian.Uint32(raw[92:]))
	for i := range methodSize {
		base := methodOff + 8*i
		classIdx := binary.LittleEndian.Uint16(raw[base:])
		protoIdx := binary.LittleEndian.Uint16(raw[base+2:])
		nameIdx := binary.LittleEndian.Uint32(raw[base+4:])
		if typeName(uint32(classIdx)) == classDesc && stringsAt[nameIdx] == name && protoMatch(int(protoIdx)) {
			return true
		}
	}
	return false
}

func dexStrings(t *testing.T, raw []byte) []string {
	t.Helper()
	stringIDs := int(binary.LittleEndian.Uint32(raw[56:]))
	stringOff := int(binary.LittleEndian.Uint32(raw[60:]))
	out := make([]string, stringIDs)
	for i := range out {
		off := int(binary.LittleEndian.Uint32(raw[stringOff+4*i:]))
		_, off = dexULEB(t, raw, off)
		end := off
		for end < len(raw) && raw[end] != 0 {
			end++
		}
		out[i] = string(raw[off:end])
	}
	return out
}
