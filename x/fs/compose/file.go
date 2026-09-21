package compose

import (
	"fmt"
	iofs "io/fs"
	"strings"

	"github.com/lewtec/lewkit/x/path"
)

// Type is the encoding of one destination path.
type Type string

const (
	TypeLines Type = "lines"
	TypeText  Type = "text"
	TypeRef   Type = "ref"
	TypeJSON  Type = "json"
	TypeTOML  Type = "toml"
	TypeYAML  Type = "yaml"
	TypeINI   Type = "ini"
	TypeXML   Type = "xml"
)

const (
	slotText = "text"
	slotRef  = "ref"
)

// Slot is one keyed fragment. Build one with [Text] or [Ref].
type Slot struct {
	kind string
	body string
}

// Text is a slot whose body is the given string. An empty body is a valid slot.
func Text(body string) Slot {
	return Slot{kind: slotText, body: body}
}

// Ref is a slot whose body is a name in the base filesystem passed to [Tree.FS].
func Ref(name string) Slot {
	return Slot{kind: slotRef, body: name}
}

func (slot Slot) valid() error {
	switch slot.kind {
	case slotText:
		return nil
	case slotRef:
		if slot.body == "" || strings.HasPrefix(slot.body, "~") {
			return fmt.Errorf("%w: %q", ErrRef, slot.body)
		}
		name := path.New(slot.body)
		if name.IsAbs() || !name.Valid() || name.String() == "." || name.String() != slot.body {
			return fmt.Errorf("%w: %q", ErrRef, slot.body)
		}
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrSlot, slot.kind)
	}
}

// File is one destination path.
// Values holds slots for [TypeLines], [TypeText], and [TypeRef].
// Data holds the map for a structured type.
// Mode 0 means unset and takes the other side during merge. The encoded
// file uses 0644 when the mode is still unset.
type File struct {
	Type   Type
	Values map[string]Slot
	Data   map[string]any
	Mode   iofs.FileMode
}

func (fileType Type) known() bool {
	switch fileType {
	case TypeLines, TypeText, TypeRef, TypeJSON, TypeTOML, TypeYAML, TypeINI, TypeXML:
		return true
	default:
		return false
	}
}

func (fileType Type) structured() bool {
	switch fileType {
	case TypeJSON, TypeTOML, TypeYAML, TypeINI, TypeXML:
		return true
	default:
		return false
	}
}

func (file File) check() error {
	if !file.Type.known() {
		return fmt.Errorf("%w: %q", ErrType, file.Type)
	}
	if file.Type.structured() {
		if file.Data == nil || len(file.Values) != 0 {
			return ErrData
		}
		return nil
	}
	if file.Data != nil {
		return ErrData
	}
	for key, slot := range file.Values {
		if key == "" {
			return fmt.Errorf("%w: empty key", ErrSlot)
		}
		if err := slot.valid(); err != nil {
			return fmt.Errorf("values.%s: %w", key, err)
		}
	}
	count := len(file.Values)
	switch file.Type {
	case TypeText:
		if count != 1 {
			return fmt.Errorf("%w: text got %d", ErrArity, count)
		}
	case TypeRef:
		if count != 1 {
			return fmt.Errorf("%w: ref got %d", ErrArity, count)
		}
		for _, slot := range file.Values {
			if slot.kind != slotRef {
				return ErrSlot
			}
		}
	}
	return nil
}

func checkName(name path.Path) error {
	text := name.String()
	if text == "" || text == "." || strings.HasPrefix(text, "~") || name.IsAbs() || !name.Valid() {
		return fmt.Errorf("%w: %q", ErrPath, text)
	}
	return nil
}

func pathClash(left, right string) bool {
	if left == right {
		return false
	}
	return strings.HasPrefix(left, right+"/") || strings.HasPrefix(right, left+"/")
}
