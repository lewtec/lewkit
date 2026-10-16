package zip

import (
	"archive/zip"
	"io"
	"io/fs"
	"path"
)

func writeArchive(f *File, w io.Writer) error {
	if err := validate(f); err != nil {
		return err
	}
	zw := zip.NewWriter(w)
	for _, entry := range f.Entries {
		if err := writeEntry(zw, entry); err != nil {
			return err
		}
	}
	return zw.Close()
}

func writeEntry(zw *zip.Writer, entry *Entry) error {
	hdr := &zip.FileHeader{
		Name:   entry.Name,
		Method: zip.Deflate,
	}
	mode := entry.Mode
	var payload []byte
	switch kind := mode & fs.ModeType; kind {
	case fs.ModeDir:
		hdr.Name += "/"
		hdr.Method = zip.Store
		mode = fs.ModeDir | permission(mode)
	case fs.ModeSymlink:
		hdr.Method = zip.Store
		mode = fs.ModeSymlink | permission(mode)
		payload = []byte(entry.Target)
	case 0:
		mode = permission(mode)
		payload = entry.Data
	default:
		return ErrEntry
	}
	hdr.SetMode(mode)
	hdr.Modified = wall(entry.Time)
	writer, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	if len(payload) == 0 {
		return nil
	}
	_, err = writer.Write(payload)
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
