package fs

import (
	iofs "io/fs"
)

// StripTopDirectory returns fsys with one leading directory removed.
// A tree that is not a single top-level directory is returned unchanged.
// Compose it in front of [Walk] and [Copy]:
//
//	stripped, err := fs.StripTopDirectory(archive)
//	err = fs.Copy(ctx, dest, fs.Walk(ctx, stripped, nil))
func StripTopDirectory(fsys iofs.FS) (iofs.FS, error) {
	entries, err := iofs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return fsys, nil
	}
	return iofs.Sub(fsys, entries[0].Name())
}
