package uki

import (
	"fmt"
	"io"
	"strings"
)

type cpioFile struct {
	name string
	mode uint32
	data []byte
}

func writeCpio(w io.Writer, files []cpioFile) error {
	var ino uint32 = 1
	for _, file := range files {
		if err := writeCpioOne(w, ino, file); err != nil {
			return err
		}
		ino++
	}
	return writeCpioOne(w, ino, cpioFile{name: "TRAILER!!!"})
}

func writeCpioOne(w io.Writer, ino uint32, file cpioFile) error {
	name := strings.TrimPrefix(file.name, "/")
	if name == "" {
		name = "."
	}
	hdr := fmt.Sprintf("070701%08X%08X%08X%08X%08X%08X%08X%08X%08X%08X%08X%08X%08X",
		ino,
		file.mode,
		uint32(0),
		uint32(0),
		uint32(1),
		uint32(0),
		uint32(len(file.data)),
		uint32(0),
		uint32(0),
		uint32(0),
		uint32(0),
		uint32(len(name)+1),
		uint32(0),
	)
	if _, err := io.WriteString(w, hdr); err != nil {
		return err
	}
	if _, err := io.WriteString(w, name); err != nil {
		return err
	}
	if _, err := w.Write([]byte{0}); err != nil {
		return err
	}
	if err := padCpio(w, len(hdr)+len(name)+1); err != nil {
		return err
	}
	if _, err := w.Write(file.data); err != nil {
		return err
	}
	return padCpio(w, len(file.data))
}

func padCpio(w io.Writer, n int) error {
	if pad := (4 - n%4) % 4; pad != 0 {
		_, err := w.Write(make([]byte, pad))
		return err
	}
	return nil
}

// cpioMode is a Unix mode with the file type bits, written in the header.
const (
	cpioTypeDir  uint32 = 0o040000
	cpioTypeFile uint32 = 0o100000
)
