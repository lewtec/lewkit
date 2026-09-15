// Package path is a pathlib-inspired name type for [io/fs].
//
// # Reader
//
// Callers that join, rewrite, or open files through [io/fs] or [*Root].
// After this file, you can pick a name, pass it to a filesystem, and
// know when a string is an OS path instead.
//
// # Two strings
//
// A [Path] holds a name. A name is unrooted and slash-separated.
// [io/fs.ValidPath] defines the legal form, except that this package
// also allows "." as the name of the filesystem root.
//
// An OS path is a host string: [os.TempDir], a CLI flag, [os.Args].
// Do not store an OS path in a Path. Open the OS path with [Open]
// to get a [*Root]. Then use Path names inside that root.
//
//	root, err := path.Open(dir) // dir is an OS path
//	p := path.New("sqlite", "q.sql")
//	b, err := p.ReadFile(root)
//
// [path.Join] from the standard library joins import paths and URLs.
// That is a third string. Leave those calls on the standard path
// package. A Path is a filesystem name only.
//
// # Pure type
//
// Path is immutable and comparable. Constructors and methods that
// return a Path do no I/O. They use the standard path package
// (slash algebra), not path/filepath.
//
// [New] joins segments with [path.Join]. Empty [New] is ".".
// [path.Join] removes "." and collapses "..". That differs from
// Python pathlib, which keeps "..". Collapsing is correct here
// because [io/fs] rejects "..".
//
// An absolute segment drops the segments before it, same as
// pathlib and [path/filepath.Join]. Standard [path.Join] keeps
// the earlier segments. The result is not a valid [io/fs] name.
// [Path.IsAbs] reports that case. Do not pass an absolute Path
// to a filesystem.
//
// # Impure calls
//
// I/O methods take an [io/fs.FS]. The Path value does not store a
// filesystem. The same Path works with [os.DirFS], [testing/fstest.MapFS],
// [embed.FS], zip, and [*Root].
//
//	p := path.New("a.txt")
//	data, err := p.ReadFile(fsys)
//
// [OpenFS] opens a name and hands the file to an adapter such as
// [github.com/lewtec/lewkit/x/fs/udf.Open]:
//
//	vol, err := path.OpenFS(path.New("en-us.iso"), root, udf.Open)
//
// Read methods use the [io/fs] helpers: [io/fs.ReadFile], [io/fs.Stat],
// [io/fs.ReadDir], [io/fs.WalkDir], [io/fs.ReadLink], [io/fs.Lstat].
// Optional interfaces on the filesystem apply as in the standard library.
//
// [Path.Glob] and [Path.Rglob] yield matches via doublestar. "**" does
// not follow symlinks. [Path.Walk] yields names under p.
//
// [Path.Select] and [Path.Under] filter a name iterator.
// [Path.MatchGlob] is doublestar against one name. A container
// listing matches with f.Name.MatchGlob; use Select/Under when
// you already have names.
//
// # Writes
//
// [io/fs.FS] is read-only. Write methods type-assert a single method
// on the filesystem (WriteFile, Mkdir, Remove, Rename, Create, …).
// There is no exported WriteFS interface.
//
// [*Root] wraps [os.OpenRoot] and implements those write methods.
// Prefer [Open] over [os.DirFS]: [os.DirFS] follows symlinks out of
// the directory. [os.OpenRoot] does not.
//
// If the filesystem does not implement the write method, the call
// returns [ErrReadOnly] through an [io/fs.PathError].
//
// # Python pathlib
//
// Path is PurePath. The filesystem argument is the concrete flavour.
// There is no Path / PurePath type split. Go has no inheritance, and
// Parent would return the wrong type.
//
// [Path.Rel] walks up (like filepath.Rel and pathlib walk_up=True).
// [Path.Symlink] and [Path.Hardlink] use pathlib order: the receiver
// is the new link, the argument is the target.
//
// Predicates return (bool, error). Missing names return false and a
// nil error. Other errors return to the caller.
//
// cwd, home, expanduser, and absolute are how you obtain a root,
// not Path methods. Pass the OS directory to [Open].
//
// This package does not implement copy/move, owner/group,
// or reserved Windows names. Name predicates live in
// [github.com/lewtec/lewkit/x/path/keep].
//
// # Out of scope
//
// Do not put t.TempDir() or "/var/lib/app" into [New]. Open that
// directory, then use names under it. x/db/generate still uses
// path/filepath for OS paths; migrate those call sites later.
package path
