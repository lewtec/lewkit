package gotool

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// packObjects appends object files to a compiler -pack archive.
// The header matches cmd/go's packInternal. The name is 16 bytes.
func packObjects(archive string, objects []string) error {
	dst, err := os.OpenFile(archive, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAsm, err)
	}
	w := bufio.NewWriter(dst)
	for _, obj := range objects {
		if err := packOne(w, obj); err != nil {
			dst.Close()
			return err
		}
	}
	if err := w.Flush(); err != nil {
		dst.Close()
		return fmt.Errorf("%w: %w", ErrAsm, err)
	}
	if err := dst.Close(); err != nil {
		return fmt.Errorf("%w: %w", ErrAsm, err)
	}
	return nil
}

func packOne(w *bufio.Writer, obj string) error {
	src, err := os.Open(obj)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAsm, err)
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAsm, err)
	}
	name := info.Name()
	if len(name) > 16 {
		name = name[:16]
	} else {
		name += strings.Repeat(" ", 16-len(name))
	}
	size := info.Size()
	if _, err := fmt.Fprintf(w, "%s%-12d%-6d%-6d%-8o%-10d`\n", name, 0, 0, 0, 0o644, size); err != nil {
		return fmt.Errorf("%w: %w", ErrAsm, err)
	}
	n, err := io.Copy(w, src)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrAsm, obj, err)
	}
	if n != size {
		return fmt.Errorf("%w: %s", ErrAsm, obj)
	}
	if size&1 != 0 {
		if err := w.WriteByte(0); err != nil {
			return fmt.Errorf("%w: %w", ErrAsm, err)
		}
	}
	return nil
}
