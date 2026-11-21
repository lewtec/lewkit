package disasm

import (
	"debug/elf"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ErrUnknownFormat is [ReadText] on bytes that are not ELF, PE, or Mach-O.
var ErrUnknownFormat = errors.New("unknown object format")

// ErrNoText is [ReadText] when the named or default text section is missing.
var ErrNoText = errors.New("no text section")

// Object is an opened object file. [ELFFile], [PEFile], and [MachOFile] implement it.
type Object interface {
	Architecture() (Architecture, Mode, error)
	TextSection(name string) (address uint64, data []byte, section string, err error)
	Symbols() Symbols
	Close() error
}

// Text is a code slice taken from an object file.
type Text struct {
	Architecture Architecture
	Mode         Mode
	Address      uint64
	Bytes        []byte
	Section      string
	Format       string
	Symbols      Symbols
}

// ReadText reads the text section of an ELF, PE, or Mach-O image.
// An empty section name selects .text, __text, or the first executable
// section, in that order.
func ReadText(reader io.ReaderAt, size int64, section string) (Text, error) {
	file, format, err := OpenObject(reader, size)
	if err != nil {
		return Text{}, err
	}
	return readObject(file, format, section)
}

// OpenObject detects ELF, PE, or Mach-O and returns the matching [Object].
func OpenObject(reader io.ReaderAt, size int64) (Object, string, error) {
	if size < 4 {
		return nil, "", ErrUnknownFormat
	}
	var magic [4]byte
	if _, err := reader.ReadAt(magic[:], 0); err != nil {
		return nil, "", err
	}
	switch {
	case string(magic[:]) == elf.ELFMAG:
		file, err := elf.NewFile(reader)
		if err != nil {
			return nil, "", fmt.Errorf("elf: %w", err)
		}
		return &ELFFile{File: file}, "elf", nil
	case magic[0] == 'M' && magic[1] == 'Z':
		file, err := openPE(reader)
		if err != nil {
			return nil, "", err
		}
		return file, "pe", nil
	case machoMagic(magic):
		file, err := openMachO(reader)
		if err != nil {
			return nil, "", err
		}
		return file, "macho", nil
	default:
		return nil, "", ErrUnknownFormat
	}
}

func readObject(file Object, format, section string) (Text, error) {
	defer file.Close()
	architecture, mode, err := file.Architecture()
	if err != nil {
		return Text{}, err
	}
	address, data, name, err := file.TextSection(section)
	if err != nil {
		return Text{}, err
	}
	return Text{
		Architecture: architecture,
		Mode:         mode,
		Address:      address,
		Bytes:        data,
		Section:      name,
		Format:       format,
		Symbols:      file.Symbols(),
	}, nil
}

func machoMagic(magic [4]byte) bool {
	value := binary.LittleEndian.Uint32(magic[:])
	switch value {
	case 0xfeedface, 0xfeedfacf, 0xcefaedfe, 0xcffaedfe:
		return true
	default:
		return false
	}
}
