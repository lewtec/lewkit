package compose

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"strings"

	"cuelang.org/go/cue"

	"github.com/lewtec/lewkit/x/path"
)

//go:embed file.cue
var schemaSource string

// Mount returns CUE that defines the destination schema and constrains
// mountPath to #Tree. mountPath is dotted, for example "app.dest".
func Mount(mountPath string) (string, error) {
	parts, err := splitCuePath(mountPath)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	builder.WriteString("_compose: {\n")
	builder.WriteString(schemaSource)
	if !strings.HasSuffix(schemaSource, "\n") {
		builder.WriteByte('\n')
	}
	builder.WriteString(structuredTypesCUE())
	builder.WriteString("}\n")
	for index, part := range parts {
		builder.WriteString(strings.Repeat("\t", index))
		if index == len(parts)-1 {
			builder.WriteString(part)
			builder.WriteString("?: _compose.#Tree\n")
			continue
		}
		builder.WriteString(part)
		builder.WriteString(": {\n")
	}
	for index := len(parts) - 2; index >= 0; index-- {
		builder.WriteString(strings.Repeat("\t", index))
		builder.WriteString("}\n")
	}
	return builder.String(), nil
}

// Constrain unifies [Mount] onto value.
func Constrain(value cue.Value, mountPath string) (cue.Value, error) {
	source, err := Mount(mountPath)
	if err != nil {
		return cue.Value{}, err
	}
	cueContext := value.Context()
	if cueContext == nil {
		return cue.Value{}, fmt.Errorf("constrain %s: %w", mountPath, errNilCue)
	}
	layer := cueContext.CompileString(source, cue.Filename("compose.cue"))
	if err := layer.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("compile mount %s: %w", mountPath, err)
	}
	out := value.Unify(layer)
	if err := out.Err(); err != nil {
		return cue.Value{}, fmt.Errorf("unify mount %s: %w", mountPath, err)
	}
	return out, nil
}

// Parse reads a #Tree value into a destination tree.
// A missing value is an empty tree.
func Parse(value cue.Value) (*Tree, error) {
	tree := New()
	if !value.Exists() {
		return tree, nil
	}
	if err := value.Err(); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if value.Kind() == cue.BottomKind && !value.IsConcrete() {
		return tree, nil
	}
	iterator, err := value.Fields()
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	for iterator.Next() {
		name := iterator.Selector().Unquoted()
		file, err := parseFile(iterator.Value())
		if err != nil {
			return nil, pathError("parse", name, err)
		}
		if err := tree.Add(path.New(name), file); err != nil {
			return nil, err
		}
	}
	return tree, nil
}

func parseFile(value cue.Value) (File, error) {
	if err := value.Err(); err != nil {
		return File{}, err
	}
	typeName, err := value.LookupPath(cue.ParsePath("type")).String()
	if err != nil {
		return File{}, fmt.Errorf("type: %w", err)
	}
	file := File{Type: Type(typeName)}
	modeValue := value.LookupPath(cue.ParsePath("mode"))
	if modeValue.Exists() {
		mode, err := modeValue.Int64()
		if err != nil || mode < 0 || mode > math.MaxUint32 {
			return File{}, ErrMode
		}
		file.Mode = fs.FileMode(mode)
	}
	values := value.LookupPath(cue.ParsePath("values"))
	if err := values.Err(); err != nil {
		return File{}, fmt.Errorf("values: %w", err)
	}
	if file.Type.structured() {
		data, err := decodeData(values)
		if err != nil {
			return File{}, err
		}
		file.Data = data
		return file, nil
	}
	iterator, err := values.Fields()
	if err != nil {
		return File{}, fmt.Errorf("values: %w", err)
	}
	slots := map[string]Slot{}
	for iterator.Next() {
		key := iterator.Selector().Unquoted()
		slot, err := parseSlot(iterator.Value())
		if err != nil {
			return File{}, fmt.Errorf("values.%s: %w", key, err)
		}
		slots[key] = slot
	}
	file.Values = slots
	return file, nil
}

func parseSlot(value cue.Value) (Slot, error) {
	if err := value.Err(); err != nil {
		return Slot{}, err
	}
	switch value.Kind() {
	case cue.StringKind:
		text, err := value.String()
		if err != nil {
			return Slot{}, err
		}
		return Text(text), nil
	case cue.StructKind:
		kind, err := value.LookupPath(cue.ParsePath("kind")).String()
		if err != nil {
			return Slot{}, fmt.Errorf("%w: missing kind", ErrSlot)
		}
		switch kind {
		case slotText:
			text, err := value.LookupPath(cue.ParsePath("text")).String()
			if err != nil {
				return Slot{}, fmt.Errorf("slot kind text: %w", err)
			}
			return Text(text), nil
		case slotRef:
			name, err := value.LookupPath(cue.ParsePath("ref")).String()
			if err != nil {
				return Slot{}, fmt.Errorf("slot kind ref: %w", err)
			}
			return Ref(name), nil
		case slotLink:
			target, err := value.LookupPath(cue.ParsePath("link")).String()
			if err != nil {
				return Slot{}, fmt.Errorf("slot kind link: %w", err)
			}
			return Link(target), nil
		default:
			return Slot{}, fmt.Errorf("%w: %s", ErrSlot, kind)
		}
	default:
		return Slot{}, fmt.Errorf("%w: want string or struct, got %s", ErrSlot, value.Kind())
	}
}

func decodeData(value cue.Value) (map[string]any, error) {
	if err := value.Err(); err != nil {
		return nil, err
	}
	if value.Kind() != cue.StructKind {
		return nil, fmt.Errorf("%w: got %s", ErrData, value.Kind())
	}
	raw, err := value.MarshalJSON()
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var data map[string]any
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}
	if data == nil {
		data = map[string]any{}
	}
	normalized, err := normalize(data)
	if err != nil {
		return nil, err
	}
	out, ok := normalized.(map[string]any)
	if !ok {
		return nil, ErrData
	}
	return out, nil
}

func splitCuePath(mountPath string) ([]string, error) {
	mountPath = strings.TrimSpace(mountPath)
	if mountPath == "" {
		return nil, fmt.Errorf("%w: empty", ErrMount)
	}
	parts := strings.Split(mountPath, ".")
	for _, part := range parts {
		if part == "" || !isCueIdent(part) {
			return nil, fmt.Errorf("%w: %q", ErrMount, mountPath)
		}
	}
	return parts, nil
}

func isCueIdent(text string) bool {
	if text == "" {
		return false
	}
	for index, char := range text {
		if index == 0 {
			if char != '_' && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') {
				return false
			}
			continue
		}
		if char != '_' && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}
