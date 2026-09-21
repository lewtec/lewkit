package fs

import (
	"context"
	iofs "io/fs"
	"strings"

	"github.com/lewtec/lewkit/x/path"
)

// StripTopDirectory returns the listing of fsys with one leading directory removed.
// A tree that is not a single top-level directory is unchanged.
// The directory member itself is dropped. Compose it in front of [Copy]:
//
//	err = fs.Copy(ctx, dest, fs.StripTopDirectory(ctx, archive))
func StripTopDirectory(ctx context.Context, fsys iofs.FS) Files {
	prefix, err := topDirectory(ctx, fsys)
	return func(yield func(File, error) bool) {
		if err != nil {
			yield(File{}, err)
			return
		}
		for file, walkErr := range Walk(ctx, fsys, nil) {
			if walkErr != nil {
				yield(File{}, walkErr)
				return
			}
			if prefix != "" {
				if file.Name.String() == prefix {
					continue
				}
				rest, ok := strings.CutPrefix(file.Name.String(), prefix+"/")
				if !ok {
					if !yield(file, nil) {
						return
					}
					continue
				}
				file.Name = path.New(rest)
			}
			if !yield(file, nil) {
				return
			}
		}
	}
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
