package udf

import (
	"encoding/binary"
	"time"
	"unicode/utf16"
)

const (
	sectorSize = 2048

	vdsSector  = 257
	vdsSectors = 16
	partSector = 288

	tagAVDP = 0x0002
	tagPVD  = 0x0001
	tagPD   = 0x0005
	tagLVD  = 0x0006
	tagTD   = 0x0008
	tagFSD  = 0x0100
	tagFID  = 0x0101
	tagFE   = 0x0105

	fileTypeDir     = 4
	fileTypeFile    = 5
	fileTypeSymlink = 12

	// fidLookahead is the Go size of github.com/Xmister/udf.FileIdentifierDescriptor.
	// ReadDir parses a descriptor only when that many bytes remain, and the
	// identifier decode reads 32 bytes at offset 38 even when implementation
	// use is empty.
	fidLookahead = 112
)

func finishTag(buf []byte, id uint16, location uint32, crcLen int) {
	binary.LittleEndian.PutUint16(buf[0:], id)
	binary.LittleEndian.PutUint16(buf[2:], 3)
	buf[4] = 0
	buf[5] = 0
	binary.LittleEndian.PutUint16(buf[6:], 0)
	binary.LittleEndian.PutUint16(buf[8:], crc16(buf[16:16+crcLen]))
	binary.LittleEndian.PutUint16(buf[10:], uint16(crcLen))
	binary.LittleEndian.PutUint32(buf[12:], location)
	var sum byte
	for i := 0; i < 16; i++ {
		if i != 4 {
			sum += buf[i]
		}
	}
	buf[4] = sum
}

func checksumOK(tag []byte) bool {
	if len(tag) < 16 {
		return false
	}
	var sum byte
	for i := 0; i < 16; i++ {
		if i != 4 {
			sum += tag[i]
		}
	}
	return sum == tag[4]
}

func crc16(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for range 8 {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func putRegID(dst []byte, id string) {
	copy(dst[1:24], id)
}

func putDString(dst []byte, s string) {
	copy(dst[:len(dst)-1], s)
	dst[len(dst)-1] = byte(len(s))
}

func readDString(b []byte) (string, error) {
	if len(b) == 0 {
		return "", ErrFormat
	}
	n := int(b[len(b)-1])
	if n > len(b)-1 {
		return "", ErrFormat
	}
	return string(b[:n]), nil
}

func putTS(b []byte, t time.Time) {
	t = wall(t)
	binary.LittleEndian.PutUint16(b[0:], 0x1000)
	binary.LittleEndian.PutUint16(b[2:], uint16(t.Year()))
	b[4] = byte(t.Month())
	b[5] = byte(t.Day())
	b[6] = byte(t.Hour())
	b[7] = byte(t.Minute())
	b[8] = byte(t.Second())
}

func readTS(b []byte) (time.Time, error) {
	if len(b) < 12 {
		return time.Time{}, ErrFormat
	}
	year := int(binary.LittleEndian.Uint16(b[2:]))
	month := int(b[4])
	day := int(b[5])
	hour := int(b[6])
	minute := int(b[7])
	second := int(b[8])
	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || minute > 59 || second > 59 {
		return time.Time{}, ErrFormat
	}
	return time.Date(year, time.Month(month), day, hour, minute, second, 0, time.UTC), nil
}

func encodeIdent(name string) ([]byte, error) {
	if name == "" {
		return nil, ErrName
	}
	for i := 0; i < len(name); i++ {
		if name[i] >= 0x80 {
			return encodeUTF16(name)
		}
	}
	if 1+len(name) > 255 {
		return nil, ErrName
	}
	out := make([]byte, 1+len(name))
	out[0] = 8
	copy(out[1:], name)
	return out, nil
}

func encodeUTF16(name string) ([]byte, error) {
	units := utf16.Encode([]rune(name))
	if 1+2*len(units) > 255 {
		return nil, ErrName
	}
	out := make([]byte, 1+2*len(units))
	out[0] = 16
	for i, unit := range units {
		binary.BigEndian.PutUint16(out[1+2*i:], unit)
	}
	return out, nil
}

func decodeIdent(b []byte) (string, error) {
	if len(b) == 0 {
		return "", ErrFormat
	}
	switch b[0] {
	case 8:
		return string(b[1:]), nil
	case 16:
		if (len(b)-1)%2 != 0 {
			return "", ErrFormat
		}
		units := make([]uint16, (len(b)-1)/2)
		for i := range units {
			units[i] = binary.BigEndian.Uint16(b[1+2*i:])
		}
		return string(utf16.Decode(units)), nil
	default:
		return "", ErrFormat
	}
}

func fidLen(identLen int) int {
	n := 38 + identLen
	return 4 * ((n + 3) / 4)
}

func encodeSymlink(target string) ([]byte, error) {
	if target == "" {
		return nil, ErrSymlink
	}
	var buf []byte
	body := target
	if body[0] == '/' {
		buf = appendComponent(buf, 1, nil)
		body = body[1:]
		if body == "" {
			return buf, nil
		}
	}
	for _, part := range splitSlash(body) {
		switch part {
		case "":
			return nil, ErrSymlink
		case ".":
			buf = appendComponent(buf, 4, nil)
		case "..":
			buf = appendComponent(buf, 3, nil)
		default:
			buf = appendComponent(buf, 5, []byte(part))
		}
	}
	return buf, nil
}

func decodeSymlink(b []byte) (string, error) {
	var parts []string
	abs := false
	for len(b) > 0 {
		if len(b) < 4 {
			return "", ErrFormat
		}
		kind := b[0]
		n := int(b[1])
		size := 4 + n
		if rem := size % 4; rem != 0 {
			size += 4 - rem
		}
		if n < 0 || size > len(b) || 4+n > len(b) {
			return "", ErrFormat
		}
		ident := string(b[4 : 4+n])
		b = b[size:]
		switch kind {
		case 1:
			abs = true
		case 3:
			parts = append(parts, "..")
		case 4:
			parts = append(parts, ".")
		case 5:
			parts = append(parts, ident)
		default:
			return "", ErrFormat
		}
	}
	joined := joinSlash(parts)
	if abs {
		return "/" + joined, nil
	}
	return joined, nil
}

func appendComponent(buf []byte, kind byte, ident []byte) []byte {
	n := 4 + len(ident)
	pad := (4 - n%4) % 4
	start := len(buf)
	buf = append(buf, make([]byte, n+pad)...)
	buf[start] = kind
	buf[start+1] = byte(len(ident))
	copy(buf[start+4:], ident)
	return buf
}

func splitSlash(s string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '/' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	return parts
}

func joinSlash(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	n := len(parts) - 1
	for _, part := range parts {
		n += len(part)
	}
	buf := make([]byte, 0, n)
	for i, part := range parts {
		if i > 0 {
			buf = append(buf, '/')
		}
		buf = append(buf, part...)
	}
	return string(buf)
}

func unixPerm(mode uint32) uint32 {
	other := mode & 7
	group := (mode >> 3) & 7
	owner := (mode >> 6) & 7
	return other | (group << 5) | (owner << 10)
}

func permFromUDF(perms uint32) uint32 {
	other := perms & 7
	group := (perms >> 5) & 7
	owner := (perms >> 10) & 7
	return other | (group << 3) | (owner << 6)
}
