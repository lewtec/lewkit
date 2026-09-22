package fs

import (
	"context"
	iofs "io/fs"
	"strings"
)

// StripTopDirectory returns fsys with one leading directory removed.
// A tree that is not a single top-level directory is returned unchanged.
// Compose it in front of [Walk] and [Copy]:
//
//	stripped, err := fs.StripTopDirectory(ctx, archive)
//	err = fs.Copy(ctx, dest, fs.Walk(ctx, stripped, nil))
func StripTopDirectory(ctx context.Context, fsys iofs.FS) (iofs.FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	prefix, err := topDirectory(ctx, fsys)
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		return fsys, nil
	}
	return iofs.Sub(fsys, prefix)
}

func topDirectory(ctx context.Context, fsys iofs.FS) (string, error) {
	tops := map[string]struct{}{}
	nested := false
	explicitDirectory := false
	err := iofs.WalkDir(fsys, ".", func(name string, entry iofs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx)
		}
		if name == "." {
			return nil
		}
		segment, _, slash := strings.Cut(name, "/")
		tops[segment] = struct{}{}
		if slash {
			nested = true
		} else if entry.IsDir() {
			explicitDirectory = true
		}
		if len(tops) > 1 {
			return iofs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(tops) != 1 || (!nested && !explicitDirectory) {
		return "", nil
	}
	for segment := range tops {
		return segment, nil
	}
	return "", nil
}
