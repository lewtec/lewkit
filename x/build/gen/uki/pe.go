package uki

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	errEmptyKernelOrInitramfs      = errs.Wrap(errs.ErrEmpty, "uki: empty kernel or initramfs")
	errSectionHeaderOverflows      = errors.New("uki: section header overflows the stub")
	errSectionTableOverflows       = errors.New("uki: section table overflows")
	errStubAlignmentZero           = errors.New("uki: stub alignment is zero")
	errStubNoRawSections           = errs.Wrap(errs.ErrNotFound, "uki: stub has no raw sections")
	errStubNoRoom                  = errs.Wrap(errs.ErrNotFound, "uki: stub has no room for UKI sections")
	errStubNotPE                   = errs.Wrap(errs.ErrInvalid, "uki: stub is not a PE")
	errStubNotPE32                 = errors.New("uki: stub is not PE32+")
	errStubOptionalHeaderOverflows = errors.New("uki: stub optional header overflows")
	errStubSectionTableOverflows   = errors.New("uki: stub section table overflows")
)

const (
	pe32Plus     = 0x20b
	peSectionLen = 40
	// Initialized data, readable. The firmware maps the section.
	peDataRead = 0x40000040
)

type peBlob struct {
	name string
	data []byte
}

// assembleUKI appends the UKI sections to a systemd stub.
// The stub serves .initrd through the Linux initrd protocol and starts .linux.
func assembleUKI(stub, kernel, initrd, cmdline, uname, osrel []byte) ([]byte, error) {
	if len(kernel) == 0 || len(initrd) == 0 {
		return nil, errEmptyKernelOrInitramfs
	}
	return peAddSections(stub, []peBlob{
		{name: ".osrel", data: osrel},
		{name: ".cmdline", data: cString(cmdline)},
		{name: ".uname", data: cString(uname)},
		{name: ".linux", data: kernel},
		{name: ".initrd", data: initrd},
	})
}

func cString(b []byte) []byte {
	out := append([]byte{}, b...)
	if len(out) == 0 || out[len(out)-1] != 0 {
		out = append(out, 0)
	}
	return out
}

func peAddSections(stub []byte, blobs []peBlob) ([]byte, error) {
	peoff, optSize, nsec, err := peHeader(stub)
	if err != nil {
		return nil, err
	}
	opt := peoff + 24
	if binary.LittleEndian.Uint16(stub[opt:]) != pe32Plus {
		return nil, errStubNotPE32
	}
	secAlign := binary.LittleEndian.Uint32(stub[opt+32:])
	fileAlign := binary.LittleEndian.Uint32(stub[opt+36:])
	sizeHeaders := binary.LittleEndian.Uint32(stub[opt+60:])
	if secAlign == 0 || fileAlign == 0 {
		return nil, errStubAlignmentZero
	}
	table := opt + int(optSize)
	firstRaw, last, err := peBounds(stub, table, int(nsec))
	if err != nil {
		return nil, err
	}
	headerEnd := table + int(nsec)*peSectionLen
	limit := int(firstRaw)
	if int(sizeHeaders) < limit {
		limit = int(sizeHeaders)
	}
	if headerEnd+len(blobs)*peSectionLen > limit {
		return nil, errStubNoRoom
	}
	out := append([]byte{}, stub...)
	// A certificate sits after the last section and is not a loaded image.
	// Drop it. The directory entry is a file offset, not a virtual address.
	end := int(last.raw + last.rsz)
	if end >= 0 && end < len(out) {
		out = out[:end]
	}
	if n := binary.LittleEndian.Uint32(out[opt+108:]); n > 4 {
		dir := opt + 112 + 4*8
		if dir+8 <= len(out) {
			binary.LittleEndian.PutUint32(out[dir:], 0)
			binary.LittleEndian.PutUint32(out[dir+4:], 0)
		}
	}
	va := alignUp(last.va+last.vsz, secAlign)
	raw := alignUp(last.raw+last.rsz, fileAlign)
	var initData uint32
	for i, blob := range blobs {
		if len(blob.name) == 0 || len(blob.name) > 8 {
			return nil, fmt.Errorf("%w: section name %q", errUKI, blob.name)
		}
		if len(blob.data) > int(^uint32(0)) {
			return nil, fmt.Errorf("%w: section %s is too large", errUKI, blob.name)
		}
		vsz := uint32(len(blob.data))
		rsz := alignUp(vsz, fileAlign)
		hdr := table + (int(nsec)+i)*peSectionLen
		if hdr+peSectionLen > len(out) {
			return nil, errSectionHeaderOverflows
		}
		for j := 0; j < peSectionLen; j++ {
			out[hdr+j] = 0
		}
		copy(out[hdr:hdr+8], blob.name)
		binary.LittleEndian.PutUint32(out[hdr+8:], vsz)
		binary.LittleEndian.PutUint32(out[hdr+12:], va)
		binary.LittleEndian.PutUint32(out[hdr+16:], rsz)
		binary.LittleEndian.PutUint32(out[hdr+20:], raw)
		binary.LittleEndian.PutUint32(out[hdr+36:], peDataRead)
		if int(raw) > len(out) {
			out = append(out, make([]byte, int(raw)-len(out))...)
		}
		if int(raw) != len(out) {
			return nil, fmt.Errorf("%w: section %s overlaps the stub", errUKI, blob.name)
		}
		out = append(out, blob.data...)
		if pad := int(rsz) - len(blob.data); pad > 0 {
			out = append(out, make([]byte, pad)...)
		}
		initData += rsz
		va = alignUp(va+vsz, secAlign)
		raw += rsz
	}
	binary.LittleEndian.PutUint16(out[peoff+4+2:], nsec+uint16(len(blobs)))
	binary.LittleEndian.PutUint32(out[opt+56:], alignUp(va, secAlign))
	binary.LittleEndian.PutUint32(out[opt+8:], binary.LittleEndian.Uint32(out[opt+8:])+initData)
	binary.LittleEndian.PutUint32(out[opt+64:], 0)
	return out, nil
}

type peLast struct {
	va, vsz, raw, rsz uint32
}

func peHeader(image []byte) (peoff int, optSize, nsec uint16, err error) {
	if len(image) < 0x40 || image[0] != 'M' || image[1] != 'Z' {
		return 0, 0, 0, errStubNotPE
	}
	peoff = int(binary.LittleEndian.Uint32(image[0x3c:]))
	if peoff < 0 || peoff+24 > len(image) || string(image[peoff:peoff+4]) != "PE\x00\x00" {
		return 0, 0, 0, errStubNotPE
	}
	nsec = binary.LittleEndian.Uint16(image[peoff+4+2:])
	optSize = binary.LittleEndian.Uint16(image[peoff+4+16:])
	if peoff+24+int(optSize) > len(image) {
		return 0, 0, 0, errStubOptionalHeaderOverflows
	}
	return peoff, optSize, nsec, nil
}

func peBounds(image []byte, table, nsec int) (firstRaw uint32, last peLast, err error) {
	if table+nsec*peSectionLen > len(image) {
		return 0, peLast{}, errStubSectionTableOverflows
	}
	for i := 0; i < nsec; i++ {
		hdr := table + i*peSectionLen
		vsz := binary.LittleEndian.Uint32(image[hdr+8:])
		va := binary.LittleEndian.Uint32(image[hdr+12:])
		rsz := binary.LittleEndian.Uint32(image[hdr+16:])
		raw := binary.LittleEndian.Uint32(image[hdr+20:])
		if raw != 0 && (firstRaw == 0 || raw < firstRaw) {
			firstRaw = raw
		}
		if raw+rsz >= last.raw+last.rsz {
			last = peLast{va: va, vsz: vsz, raw: raw, rsz: rsz}
		}
	}
	if firstRaw == 0 {
		return 0, peLast{}, errStubNoRawSections
	}
	return firstRaw, last, nil
}

func peSection(image []byte, name string) ([]byte, error) {
	peoff, optSize, nsec, err := peHeader(image)
	if err != nil {
		return nil, err
	}
	table := peoff + 24 + int(optSize)
	if table+int(nsec)*peSectionLen > len(image) {
		return nil, errSectionTableOverflows
	}
	var out []byte
	found := false
	for i := 0; i < int(nsec); i++ {
		hdr := table + i*peSectionLen
		secName := stringsTrim(image[hdr : hdr+8])
		if secName != name {
			continue
		}
		vsz := binary.LittleEndian.Uint32(image[hdr+8:])
		raw := binary.LittleEndian.Uint32(image[hdr+20:])
		if int(raw) > len(image) || int(raw)+int(vsz) > len(image) {
			return nil, fmt.Errorf("%w: section %s overflows", errUKI, name)
		}
		out = append(out, image[raw:raw+vsz]...)
		found = true
	}
	if !found {
		return nil, fmt.Errorf("%w: section %s is missing", errUKI, name)
	}
	return out, nil
}

func stringsTrim(b []byte) string {
	i := 0
	for i < len(b) && b[i] != 0 {
		i++
	}
	return string(b[:i])
}

func alignUp(n, a uint32) uint32 {
	if a == 0 {
		return n
	}
	return (n + a - 1) &^ (a - 1)
}

func imageArch(image []byte) (string, error) {
	peoff, _, _, err := peHeader(image)
	if err != nil {
		return "", err
	}
	switch binary.LittleEndian.Uint16(image[peoff+4:]) {
	case 0x8664:
		return "amd64", nil
	case 0xaa64:
		return "arm64", nil
	default:
		return "", fmt.Errorf("%w: pe machine %#x", ErrArch, binary.LittleEndian.Uint16(image[peoff+4:]))
	}
}
