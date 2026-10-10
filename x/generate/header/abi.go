package header

import (
	"fmt"
	"runtime"
)

// Target is the C data layout the generator matches.
type Target struct {
	ID         string
	Ptr        int
	Int        int
	Long       int
	LongDouble int
	LongAlign  int
	Wchar      int
	WcharSign  bool
	Linux      bool
	Darwin     bool
	Windows    bool
	AMD64      bool
	ARM64      bool
	I386       bool
	ARM        bool
}

// SysV64 is the LP64 layout used by amd64 and arm64 on Unix.
// long double is 16 bytes. The Go type cannot be aligned to 16; the
// generator keeps the C size and pads fields to the C offsets.
func SysV64() Target {
	return Target{
		ID: "sysv64", Ptr: 8, Int: 4, Long: 8,
		LongDouble: 16, LongAlign: 16, Wchar: 4, WcharSign: true,
	}
}

// Win64 is the LLP64 layout. long is 4 bytes and long double is 8.
func Win64() Target {
	return Target{
		ID: "win64", Ptr: 8, Int: 4, Long: 4,
		LongDouble: 8, LongAlign: 8, Wchar: 2, WcharSign: false,
		Windows: true,
	}
}

// I386 is the Unix 32-bit layout. long double is 12 bytes, aligned to 4.
func I386() Target {
	return Target{
		ID: "386", Ptr: 4, Int: 4, Long: 4,
		LongDouble: 12, LongAlign: 4, Wchar: 4, WcharSign: true,
		I386: true,
	}
}

// ARM is the Unix 32-bit ARM layout.
func ARM() Target {
	return Target{
		ID: "arm", Ptr: 4, Int: 4, Long: 4,
		LongDouble: 8, LongAlign: 8, Wchar: 4, WcharSign: false,
		ARM: true,
	}
}

// Host is the C layout of this process.
func Host() Target {
	t := SysV64()
	switch runtime.GOARCH {
	case "386":
		t = I386()
	case "arm":
		t = ARM()
	case "arm64":
		t.ARM64 = true
	default:
		t.AMD64 = true
	}
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			t = Win64()
			t.AMD64 = runtime.GOARCH == "amd64"
			t.ARM64 = runtime.GOARCH == "arm64"
		}
	case "darwin", "ios":
		t.Darwin = true
	default:
		t.Linux = runtime.GOOS == "linux" || runtime.GOOS == "android"
	}
	if runtime.GOARCH == "amd64" {
		t.AMD64 = true
	}
	if runtime.GOARCH == "arm64" {
		t.ARM64 = true
	}
	return t
}

// ParseABI reads host, sysv64, win64, 386, or arm. An empty name is Host.
func ParseABI(name string) (Target, error) {
	switch name {
	case "", "host":
		return Host(), nil
	case "sysv64":
		t := SysV64()
		t.Linux = true
		t.AMD64 = true
		return t, nil
	case "win64":
		t := Win64()
		t.AMD64 = true
		return t, nil
	case "386":
		t := I386()
		t.Linux = true
		return t, nil
	case "arm":
		t := ARM()
		t.Linux = true
		return t, nil
	default:
		return Target{}, fmt.Errorf("%w: %s", errABI, name)
	}
}

// Constraint is the //go:build line for files generated for t.
func (t Target) Constraint() string {
	switch t.ID {
	case "win64":
		return "windows && (amd64 || arm64)"
	case "386":
		return "(android || darwin || freebsd || linux || netbsd || openbsd) && 386"
	case "arm":
		return "(android || linux) && arm"
	default:
		return "(aix || android || darwin || dragonfly || freebsd || illumos || ios || linux || netbsd || openbsd || solaris) && (amd64 || arm64)"
	}
}

func alignUp(n, a int) int {
	if a <= 1 {
		return n
	}
	return (n + a - 1) &^ (a - 1)
}

type cursor struct {
	bits  int
	align int
}

func (c *cursor) note(align int) {
	if align > c.align {
		c.align = align
	}
}

func (c *cursor) placeScalar(size, align int) int {
	if align < 1 {
		align = 1
	}
	byte := alignUp((c.bits+7)/8, align)
	c.bits = (byte + size) * 8
	c.note(align)
	return byte
}

// placeBits packs a bit-field into an aligned container of the declared
// type, matching the System V rule that a bit-field stays inside one
// storage unit and may share that unit with a following member.
func (c *cursor) placeBits(width, size, align int) (container, bitOff int) {
	if align < 1 {
		align = 1
	}
	if size < 1 {
		size = 1
	}
	if width == 0 {
		c.bits = alignUp(c.bits, align*8)
		c.note(align)
		return 0, 0
	}
	cbits := size * 8
	abits := align * 8
	start := c.bits
	base := start / abits * abits
	if start+width <= base+cbits {
		c.bits = start + width
		c.note(align)
		return base / 8, start - base
	}
	next := base + abits
	c.bits = next + width
	c.note(align)
	return next / 8, 0
}

func (c *cursor) end(extra int) (int, int) {
	if extra > c.align {
		c.align = extra
	}
	if c.align < 1 {
		c.align = 1
	}
	return alignUp((c.bits+7)/8, c.align), c.align
}

func (t Target) layout(typ *Type) error {
	if typ == nil || typ.laid {
		return nil
	}
	if typ.laying {
		return fmt.Errorf("%w: incomplete %s", errLayout, typ.Spell)
	}
	typ.laying = true
	defer func() { typ.laying = false }()
	var err error
	switch typ.Kind {
	case kindStruct:
		err = t.layoutStruct(typ)
	case kindUnion:
		err = t.layoutUnion(typ)
	case kindArray:
		err = t.layoutArray(typ)
	case kindAlias:
		if err = t.layout(typ.Elem); err != nil {
			return err
		}
		typ.Size = typ.Elem.Size
		typ.Align = typ.Elem.Align
	case kindPtr, kindFunc:
		typ.Size = t.Ptr
		typ.Align = t.Ptr
	case kindVoid:
		typ.Size = 0
		typ.Align = 1
	default:
		if typ.Align < 1 {
			typ.Align = 1
		}
	}
	if err != nil {
		return err
	}
	typ.laid = true
	return nil
}

func (t Target) layoutArray(typ *Type) error {
	if err := t.layout(typ.Elem); err != nil {
		return err
	}
	typ.Align = typ.Elem.Align
	if typ.Align < 1 {
		typ.Align = 1
	}
	if typ.Len < 0 {
		typ.Size = 0
		return nil
	}
	typ.Size = typ.Len * typ.Elem.Size
	return nil
}

func (t Target) fieldAlign(f *Field, packed bool) int {
	a := 1
	if f.Type != nil && f.Type.Align > 0 {
		a = f.Type.Align
	}
	if packed {
		a = 1
	}
	if f.AlignAttr > a {
		a = f.AlignAttr
	}
	return a
}

func (t Target) layoutStruct(typ *Type) error {
	if typ.Incomplete {
		return fmt.Errorf("%w: incomplete %s", errLayout, typ.Spell)
	}
	cur := cursor{align: 1}
	for i, f := range typ.Fields {
		if f.Type != nil && !f.IsBit {
			if err := t.layout(f.Type); err != nil {
				return err
			}
		} else if f.Type != nil {
			if err := t.layout(f.Type); err != nil {
				return err
			}
		}
		al := t.fieldAlign(f, typ.Packed || f.Packed)
		if f.IsBit {
			if f.BitWidth < 0 || (f.Type != nil && f.BitWidth > f.Type.Size*8) {
				return fmt.Errorf("%w: bit-field %s", errLayout, f.Name)
			}
			sz := 1
			if f.Type != nil && f.Type.Size > 0 {
				sz = f.Type.Size
			}
			cOff, bOff := cur.placeBits(f.BitWidth, sz, al)
			f.Container = cOff
			f.BitOff = bOff
			f.CSize = sz
			continue
		}
		if f.Type != nil && f.Type.flex() {
			if i != len(typ.Fields)-1 {
				return fmt.Errorf("%w: flexible array %s", errLayout, f.Name)
			}
			off := cur.placeScalar(0, al)
			f.Offset = off
			f.CSize = 0
			continue
		}
		sz := 0
		if f.Type != nil {
			sz = f.Type.Size
		}
		f.Offset = cur.placeScalar(sz, al)
		f.CSize = sz
	}
	typ.Size, typ.Align = cur.end(typ.AlignAttr)
	return nil
}

func (t Target) layoutUnion(typ *Type) error {
	if typ.Incomplete {
		return fmt.Errorf("%w: incomplete %s", errLayout, typ.Spell)
	}
	size, align := 0, 1
	for _, f := range typ.Fields {
		if err := t.layout(f.Type); err != nil {
			return err
		}
		al := t.fieldAlign(f, typ.Packed || f.Packed)
		sz := 0
		if f.Type != nil {
			sz = f.Type.Size
		}
		if f.IsBit && f.Type != nil {
			sz = f.Type.Size
		}
		if sz > size {
			size = sz
		}
		if al > align {
			align = al
		}
		f.Offset = 0
		f.CSize = sz
		f.Container = 0
		f.BitOff = 0
	}
	if typ.AlignAttr > align {
		align = typ.AlignAttr
	}
	typ.Align = align
	typ.Size = alignUp(size, align)
	return nil
}
