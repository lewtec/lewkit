package clib

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	peFileDLL      = 0x2000
	peMachine386   = 0x14c
	peMachineAmd64 = 0x8664
	peMachineArm64 = 0xaa64
	peOptPE32      = 0x10b
	peOptPE32Plus  = 0x20b
)

// MakeDLL marks an internal-link Windows PIE as a DLL and points its
// entry at a stub that returns TRUE. The Go linker already wrote the
// export table from the cgo_export_dynamic directives. The process entry
// would otherwise run as DllMain and start the runtime under the loader lock.
func MakeDLL(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(buf) < 0x40 {
		return fmt.Errorf("clib: %s is not a PE file", path)
	}
	lfanew := int(binary.LittleEndian.Uint32(buf[0x3c:]))
	if lfanew < 0 || lfanew+24 > len(buf) || string(buf[lfanew:lfanew+4]) != "PE\x00\x00" {
		return fmt.Errorf("clib: %s is not a PE file", path)
	}
	coff := lfanew + 4
	machine := binary.LittleEndian.Uint16(buf[coff:])
	optSize := int(binary.LittleEndian.Uint16(buf[coff+16:]))
	chars := binary.LittleEndian.Uint16(buf[coff+18:])
	chars |= peFileDLL
	binary.LittleEndian.PutUint16(buf[coff+18:], chars)
	opt := coff + 20
	if opt+optSize > len(buf) || opt+20 > len(buf) {
		return fmt.Errorf("clib: %s has a short optional header", path)
	}
	magic := binary.LittleEndian.Uint16(buf[opt:])
	if magic != peOptPE32 && magic != peOptPE32Plus {
		return fmt.Errorf("clib: %s has an unknown PE magic", path)
	}
	entry := binary.LittleEndian.Uint32(buf[opt+16:])
	nsects := int(binary.LittleEndian.Uint16(buf[coff+2:]))
	sect := opt + optSize
	if sect+nsects*40 > len(buf) {
		return fmt.Errorf("clib: %s section headers are out of range", path)
	}
	stub := dllStub(machine)
	if stub == nil {
		return fmt.Errorf("clib: %s is not a supported PE machine", path)
	}
	raw, ok := peFileOff(buf[sect:], nsects, entry)
	if !ok || raw < 0 || raw+len(stub) > len(buf) {
		return fmt.Errorf("clib: %s entry point is outside its sections", path)
	}
	copy(buf[raw:], stub)
	return os.WriteFile(path, buf, info.Mode())
}

func dllStub(machine uint16) []byte {
	switch machine {
	case peMachineAmd64:
		return []byte{0xb8, 0x01, 0x00, 0x00, 0x00, 0xc3}
	case peMachine386:
		// mov eax, 1; ret 12. DllMain is stdcall with three arguments.
		return []byte{0xb8, 0x01, 0x00, 0x00, 0x00, 0xc2, 0x0c, 0x00}
	case peMachineArm64:
		// movz w0, #1; ret
		return []byte{0x20, 0x00, 0x80, 0x52, 0xc0, 0x03, 0x5f, 0xd6}
	default:
		return nil
	}
}

func peFileOff(sects []byte, n int, rva uint32) (int, bool) {
	for i := 0; i < n; i++ {
		off := i * 40
		vsize := binary.LittleEndian.Uint32(sects[off+8:])
		vaddr := binary.LittleEndian.Uint32(sects[off+12:])
		rawSize := binary.LittleEndian.Uint32(sects[off+16:])
		rawPtr := binary.LittleEndian.Uint32(sects[off+20:])
		span := vsize
		if rawSize > span {
			span = rawSize
		}
		if rva < vaddr || rva >= vaddr+span {
			continue
		}
		return int(rawPtr + (rva - vaddr)), true
	}
	return 0, false
}
