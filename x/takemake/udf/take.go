package udf

import (
	"encoding/binary"
	"io/fs"
	"time"
)

func take(raw []byte) (*File, error) {
	sector, err := findSector(raw)
	if err != nil {
		return nil, err
	}
	if sector != sectorSize {
		return nil, ErrFormat
	}
	label, partStart, blockSize, fsdLBN, err := readVDS(raw)
	if err != nil {
		return nil, err
	}
	if blockSize != sectorSize {
		return nil, ErrFormat
	}
	fsd := int(partStart+fsdLBN) * sectorSize
	if err := need(raw, fsd, 464); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint16(raw[fsd:]) != tagFSD || !checksumOK(raw[fsd:fsd+16]) {
		return nil, ErrFormat
	}
	if binary.LittleEndian.Uint32(raw[fsd+12:]) != fsdLBN {
		return nil, ErrFormat
	}
	rootLBN := binary.LittleEndian.Uint32(raw[fsd+404:])
	if binary.LittleEndian.Uint16(raw[fsd+408:]) != 0 {
		return nil, ErrFormat
	}
	entries, err := walk(raw, partStart, rootLBN, "")
	if err != nil {
		return nil, err
	}
	return &File{Label: label, Entries: entries, cur: -1}, nil
}

func findSector(raw []byte) (int, error) {
	for size := 512; size <= 32768; size <<= 1 {
		off := 256 * size
		if off+32 > len(raw) {
			continue
		}
		if binary.LittleEndian.Uint16(raw[off:]) != tagAVDP || !checksumOK(raw[off:off+16]) {
			continue
		}
		return size, nil
	}
	return 0, ErrFormat
}

func readVDS(raw []byte) (label string, partStart, blockSize, fsdLBN uint32, err error) {
	anchor := 256 * sectorSize
	if err = need(raw, anchor, 32); err != nil {
		return "", 0, 0, 0, err
	}
	mainLen := binary.LittleEndian.Uint32(raw[anchor+16:])
	mainLoc := binary.LittleEndian.Uint32(raw[anchor+20:])
	if mainLen == 0 || mainLen%sectorSize != 0 || mainLoc == 0 {
		return "", 0, 0, 0, ErrFormat
	}
	sectors := mainLen / sectorSize
	if sectors > 64 {
		sectors = 64
	}
	var havePVD, havePD, haveLVD bool
	var partNumber uint16
	for i := uint32(0); i < sectors; i++ {
		at := int(mainLoc+i) * sectorSize
		if err = need(raw, at, 470); err != nil {
			return "", 0, 0, 0, err
		}
		if !checksumOK(raw[at : at+16]) {
			return "", 0, 0, 0, ErrFormat
		}
		if binary.LittleEndian.Uint32(raw[at+12:]) != mainLoc+i {
			return "", 0, 0, 0, ErrFormat
		}
		switch binary.LittleEndian.Uint16(raw[at:]) {
		case tagPVD:
			label, err = readDString(raw[at+24 : at+56])
			if err != nil {
				return "", 0, 0, 0, err
			}
			havePVD = true
		case tagPD:
			partNumber = binary.LittleEndian.Uint16(raw[at+22:])
			partStart = binary.LittleEndian.Uint32(raw[at+188:])
			havePD = true
		case tagLVD:
			blockSize = binary.LittleEndian.Uint32(raw[at+212:])
			fsdLBN = binary.LittleEndian.Uint32(raw[at+252:])
			if binary.LittleEndian.Uint16(raw[at+256:]) != 0 {
				return "", 0, 0, 0, ErrFormat
			}
			if binary.LittleEndian.Uint32(raw[at+264:]) != 6 || binary.LittleEndian.Uint32(raw[at+268:]) != 1 {
				return "", 0, 0, 0, ErrFormat
			}
			if raw[at+440] != 1 || raw[at+441] != 6 {
				return "", 0, 0, 0, ErrFormat
			}
			mapped := binary.LittleEndian.Uint16(raw[at+444:])
			if havePD && mapped != partNumber {
				return "", 0, 0, 0, ErrFormat
			}
			partNumber = mapped
			haveLVD = true
		case tagTD:
			if !havePVD || !havePD || !haveLVD {
				return "", 0, 0, 0, ErrFormat
			}
			return label, partStart, blockSize, fsdLBN, nil
		}
	}
	return "", 0, 0, 0, ErrFormat
}

func walk(raw []byte, partStart, lbn uint32, prefix string) ([]*Entry, error) {
	fe, err := readFE(raw, partStart, lbn)
	if err != nil {
		return nil, err
	}
	if fe.fileType != fileTypeDir {
		return nil, ErrFormat
	}
	fids, err := parseFIDs(fe.data)
	if err != nil {
		return nil, err
	}
	var out []*Entry
	for _, fid := range fids {
		if fid.ch&0x08 != 0 || fid.ch&0x04 != 0 || fid.name == "" {
			continue
		}
		child, err := readFE(raw, partStart, fid.lbn)
		if err != nil {
			return nil, err
		}
		name := fid.name
		if prefix != "" {
			name = prefix + "/" + fid.name
		}
		entry := &Entry{
			Name: name,
			Mode: fs.FileMode(permFromUDF(child.perm)),
			Time: child.when,
		}
		switch child.fileType {
		case fileTypeDir:
			entry.Mode |= fs.ModeDir
			out = append(out, entry)
			nested, err := walk(raw, partStart, fid.lbn, name)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
		case fileTypeSymlink:
			target, err := decodeSymlink(child.data)
			if err != nil {
				return nil, err
			}
			entry.Mode |= fs.ModeSymlink
			entry.Target = target
			out = append(out, entry)
		case fileTypeFile:
			if len(child.data) > 0 {
				entry.Data = child.data
			}
			out = append(out, entry)
		default:
			return nil, ErrKind
		}
	}
	return out, nil
}

type feView struct {
	fileType byte
	perm     uint32
	when     time.Time
	data     []byte
}

func readFE(raw []byte, partStart, lbn uint32) (feView, error) {
	off := int(partStart+lbn) * sectorSize
	if err := need(raw, off, 176); err != nil {
		return feView{}, err
	}
	b := raw[off:]
	if binary.LittleEndian.Uint16(b) != tagFE || !checksumOK(b[:16]) {
		return feView{}, ErrFormat
	}
	if binary.LittleEndian.Uint32(b[12:]) != lbn {
		return feView{}, ErrFormat
	}
	if binary.LittleEndian.Uint16(b[34:])&0x3 != 0 {
		return feView{}, ErrFormat
	}
	when, err := readTS(b[84:])
	if err != nil {
		return feView{}, err
	}
	infoLen := binary.LittleEndian.Uint64(b[56:])
	eaLen := binary.LittleEndian.Uint32(b[168:])
	adLen := binary.LittleEndian.Uint32(b[172:])
	if eaLen != 0 || (adLen != 0 && adLen != 8) {
		return feView{}, ErrFormat
	}
	if err := need(raw, off+176, int(adLen)); err != nil {
		return feView{}, err
	}
	var recorded uint32
	var dataLBN uint32
	if adLen == 8 {
		recorded = binary.LittleEndian.Uint32(b[176:])
		if recorded>>30 != 0 {
			return feView{}, ErrFormat
		}
		dataLBN = binary.LittleEndian.Uint32(b[180:])
	}
	if infoLen > uint64(recorded) {
		return feView{}, ErrFormat
	}
	var data []byte
	if infoLen > 0 {
		dataAt := int(partStart+dataLBN) * sectorSize
		if err := need(raw, dataAt, int(infoLen)); err != nil {
			return feView{}, err
		}
		data = append([]byte(nil), raw[dataAt:dataAt+int(infoLen)]...)
	}
	return feView{
		fileType: b[27],
		perm:     binary.LittleEndian.Uint32(b[44:]),
		when:     when,
		data:     data,
	}, nil
}

type fidView struct {
	ch   byte
	name string
	lbn  uint32
}

func parseFIDs(payload []byte) ([]fidView, error) {
	var out []fidView
	for off := 0; off < len(payload); {
		if off+38 > len(payload) {
			return nil, ErrFormat
		}
		rec := payload[off:]
		if binary.LittleEndian.Uint16(rec) != tagFID || !checksumOK(rec[:16]) {
			return nil, ErrFormat
		}
		nameLen := int(rec[19])
		implLen := int(binary.LittleEndian.Uint16(rec[36:]))
		identAt := 38 + implLen
		if implLen < 0 || nameLen < 0 || identAt+nameLen > len(payload)-off {
			return nil, ErrFormat
		}
		var name string
		var err error
		if nameLen > 0 {
			name, err = decodeIdent(rec[identAt : identAt+nameLen])
			if err != nil {
				return nil, err
			}
		}
		size := 38 + implLen + nameLen
		pad := 4 * ((size + 3) / 4)
		if off+pad > len(payload) {
			return nil, ErrFormat
		}
		out = append(out, fidView{
			ch:   rec[18],
			name: name,
			lbn:  binary.LittleEndian.Uint32(rec[24:]),
		})
		off += pad
	}
	return out, nil
}

func need(raw []byte, off, n int) error {
	if off < 0 || n < 0 || off > len(raw) || n > len(raw)-off {
		return ErrFormat
	}
	return nil
}
