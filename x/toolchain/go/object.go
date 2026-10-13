package gotool

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

var (
	// ErrTarget means an object was built for a different GOOS or GOARCH
	// than the link.
	ErrTarget = fmt.Errorf("toolchain: mixed target")
	// ErrEmptyTarget means the link has no GOOS, GOARCH, or mode.
	ErrEmptyTarget = fmt.Errorf("toolchain: empty target")
	// ErrNoAsm means a unit lists assembly or system objects. cmd/asm is
	// not vendored, so this toolchain does not produce that archive.
	ErrNoAsm = fmt.Errorf("toolchain: assembly is not produced here")
)

// Object is one package archive. Go source, assembly, a prebuilt runtime
// archive, and Go generated from a C header all end as this value. The
// link reads ImportPath and Archive. GOOS and GOARCH are the target that
// produced the archive. Empty GOOS and GOARCH inherit the [Target].
type Object struct {
	ImportPath string
	Archive    string
	GOOS       string
	GOARCH     string
}

// Target is one link. Runtime is the prebuilt archives for that GOOS and
// GOARCH, including the standard library. Ldflags are extra linker
// arguments. A tamago image passes [github.com/lewtec/lewkit/x/toolchain/tamago.LinkFlags].
// Mode is the linker -buildmode, such as exe or pie.
type Target struct {
	GOOS    string
	GOARCH  string
	Mode    string
	Runtime []Object
	Ldflags []string
}

// Object is the archive this unit produces.
func (u Unit) Object() Object {
	return Object{
		ImportPath: u.ImportPath,
		Archive:    u.Archive,
		GOOS:       u.GOOS,
		GOARCH:     u.GOARCH,
	}
}

// ImportCfg is the linker and compiler importcfg body for runtime archives
// followed by objs. A later archive for the same import path replaces an
// earlier one only when the path is the same file. Two different files for
// one import path are [ErrDuplicate]. An object whose GOOS or GOARCH
// differs from the target is [ErrTarget].
func (t Target) ImportCfg(objs []Object) (string, error) {
	if err := t.check(); err != nil {
		return "", err
	}
	all := make([]Object, 0, len(t.Runtime)+len(objs))
	all = append(all, t.Runtime...)
	all = append(all, objs...)
	files := make(map[string]string, len(all))
	for _, o := range all {
		if o.ImportPath == "" || o.Archive == "" {
			return "", fmt.Errorf("%w: %s", ErrEmptyUnit, o.ImportPath)
		}
		if err := t.match(o); err != nil {
			return "", err
		}
		if prev, ok := files[o.ImportPath]; ok && prev != o.Archive {
			return "", fmt.Errorf("%w: %s", ErrDuplicate, o.ImportPath)
		}
		files[o.ImportPath] = o.Archive
	}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("# import config\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "packagefile %s=%s\n", k, files[k])
	}
	return b.String(), nil
}

// LinkImage is the link step for root. root is the main archive.
func (t Target) LinkImage(out, importCfg string, root Object) (Image, error) {
	if err := t.check(); err != nil {
		return Image{}, err
	}
	if err := t.match(root); err != nil {
		return Image{}, err
	}
	if out == "" || importCfg == "" || root.Archive == "" {
		return Image{}, ErrEmptyImage
	}
	return Image{
		Out:       out,
		ImportCfg: importCfg,
		Mode:      t.Mode,
		Archive:   root.Archive,
		Ldflags:   append([]string(nil), t.Ldflags...),
		GOOS:      t.GOOS,
		GOARCH:    t.GOARCH,
	}, nil
}

func (t Target) check() error {
	if t.GOOS == "" || t.GOARCH == "" || t.Mode == "" {
		return ErrEmptyTarget
	}
	for _, o := range t.Runtime {
		if err := t.match(o); err != nil {
			return err
		}
	}
	return nil
}

func (t Target) match(o Object) error {
	if o.GOOS == "" && o.GOARCH == "" {
		return nil
	}
	if o.GOOS != t.GOOS || o.GOARCH != t.GOARCH {
		return fmt.Errorf("%w: %s is %s/%s", ErrTarget, o.ImportPath, o.GOOS, o.GOARCH)
	}
	return nil
}

// BuildObjects compiles units and links out for target.
// The unit whose import path is main is the root object. Each unit's
// GOOS and GOARCH, when set, must match target. Assembly on a unit is
// [ErrNoAsm]. The importcfg lists target.Runtime and then the units.
func BuildObjects(ctx context.Context, units []Unit, target Target, out string) error {
	if ctx == nil {
		return ErrNilContext
	}
	objs := make([]Object, 0, len(units))
	var root *Object
	prepared := make([]Unit, len(units))
	for i, u := range units {
		if u.GOOS == "" {
			u.GOOS = target.GOOS
		}
		if u.GOARCH == "" {
			u.GOARCH = target.GOARCH
		}
		prepared[i] = u
		o := u.Object()
		if u.ImportPath == "main" {
			if root != nil {
				return fmt.Errorf("%w: main", ErrDuplicate)
			}
			root = &o
		}
		objs = append(objs, o)
	}
	if root == nil {
		return fmt.Errorf("%w: main", ErrEmptyImage)
	}
	body, err := target.ImportCfg(objs)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp("", "lewkit-objects-*.importcfg")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	image, err := target.LinkImage(out, name, *root)
	if err != nil {
		return err
	}
	return Build(ctx, prepared, &image)
}
