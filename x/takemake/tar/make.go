package tar

import (
	"archive/tar"
	"errors"
	"io"
	"io/fs"
	"path"
)

func writeArchive(f *File, w io.Writer) error {
	if err := validate(f); err != nil {
		return err
	}
	tw := tar.NewWriter(w)
	for _, entry := range f.Entries {
		if err := writeEntry(tw, entry); err != nil {
			return err
		}
	}
	return tw.Close()
}

func writeEntry(tw *tar.Writer, entry *Entry) error {
	when := wall(entry.Time)
	hdr := &tar.Header{
		Name:    entry.Name,
		Mode:    tarMode(entry.Mode),
		ModTime: when,
	}
	if when.Nanosecond() != 0 {
		hdr.Format = tar.FormatPAX
	}
	var payload []byte
	switch kind := entry.Mode & fs.ModeType; kind {
	case fs.ModeDir:
		hdr.Typeflag = tar.TypeDir
	case fs.ModeSymlink:
		hdr.Typeflag = tar.TypeSymlink
		hdr.Linkname = entry.Target
	case 0:
		hdr.Typeflag = tar.TypeReg
		hdr.Size = int64(len(entry.Data))
		payload = entry.Data
	default:
		return ErrEntry
	}
	if err := tw.WriteHeader(hdr); err != nil {
		if errors.Is(err, tar.ErrFieldTooLong) {
			return ErrName
		}
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err := tw.Write(payload)
	return err
}

func validate(f *File) error {
	seen := make(map[string]fs.FileMode, len(f.Entries))
	for _, entry := range f.Entries {
		if entry == nil {
			return ErrKind
		}
		if err := checkEntry(entry); err != nil {
			return err
		}
		parent := path.Dir(entry.Name)
		if parent != "." {
			if mode, ok := seen[parent]; ok && mode&fs.ModeDir == 0 {
				return ErrKind
			}
		}
		seen[entry.Name] = entry.Mode
	}
	return nil
}

func checkEntry(entry *Entry) error {
	cleaned, err := cleanName(entry.Name)
	if err != nil || cleaned != entry.Name {
		return ErrName
	}
	dir := entry.Mode&fs.ModeDir != 0
	link := entry.Mode&fs.ModeSymlink != 0
	switch {
	case dir && link:
		return ErrKind
	case dir && (len(entry.Data) > 0 || entry.Target != ""):
		return ErrKind
	case link && (entry.Target == "" || len(entry.Data) > 0):
		return ErrKind
	case !dir && !link && entry.Target != "":
		return ErrKind
	}
	kind := entry.Mode & fs.ModeType
	if kind != 0 && kind != fs.ModeDir && kind != fs.ModeSymlink {
		return ErrEntry
	}
	return nil
}

func tarMode(mode fs.FileMode) int64 {
	m := int64(mode & 0o777)
	if mode&fs.ModeSetuid != 0 {
		m |= 0o4000
	}
	if mode&fs.ModeSetgid != 0 {
		m |= 0o2000
	}
	if mode&fs.ModeSticky != 0 {
		m |= 0o1000
	}
	return m
}
