package compose

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

// Format encodes one structured map. [Register] stores it under a type name.
type Format func(data map[string]any) ([]byte, error)

var (
	formatMu sync.RWMutex
	formats  = map[Type]Format{
		TypeJSON: encodeJSON,
		TypeTOML: encodeTOML,
		TypeYAML: encodeYAML,
		TypeINI:  encodeINI,
		TypeXML:  encodeXML,
	}
)

// Register adds a structured format under name.
// lines, text, and ref are slot types and cannot be registered.
// [Mount] includes every registered name in #StructuredType.
func Register(name Type, format Format) error {
	switch name {
	case "", TypeLines, TypeText, TypeRef:
		return fmt.Errorf("%w: %q", ErrType, name)
	}
	if format == nil {
		return fmt.Errorf("%w: nil format", ErrType)
	}
	formatMu.Lock()
	defer formatMu.Unlock()
	if _, exists := formats[name]; exists {
		return fmt.Errorf("%w: %q", ErrRegistered, name)
	}
	formats[name] = format
	return nil
}

// Formats returns the registered structured type names, sorted.
func Formats() []Type {
	formatMu.RLock()
	defer formatMu.RUnlock()
	return slices.Sorted(maps.Keys(formats))
}

func lookupFormat(name Type) (Format, bool) {
	formatMu.RLock()
	defer formatMu.RUnlock()
	format, ok := formats[name]
	return format, ok
}

func encodeStructured(fileType Type, data map[string]any) ([]byte, error) {
	format, ok := lookupFormat(fileType)
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrType, fileType)
	}
	if data == nil {
		data = map[string]any{}
	}
	body, err := format(data)
	if err != nil {
		return nil, err
	}
	return ensureNewline(body), nil
}

func encodeJSON(data map[string]any) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

func encodeTOML(data map[string]any) ([]byte, error) {
	return toml.Marshal(data)
}

func encodeYAML(data map[string]any) ([]byte, error) {
	return yaml.Marshal(data)
}

// structuredTypesCUE is a #StructuredType disjunction of the registered names.
func structuredTypesCUE() string {
	names := Formats()
	if len(names) == 0 {
		return "#StructuredType: string\n"
	}
	quoted := make([]string, len(names))
	for index, name := range names {
		quoted[index] = strconv.Quote(string(name))
	}
	return "#StructuredType: " + strings.Join(quoted, " | ") + "\n"
}

func ensureNewline(body []byte) []byte {
	if len(body) == 0 || bytes.HasSuffix(body, []byte("\n")) {
		return body
	}
	return append(body, '\n')
}

func sortedKeys(data map[string]any) []string {
	return slices.Sorted(maps.Keys(data))
}

func encodeINI(data map[string]any) ([]byte, error) {
	var buffer bytes.Buffer
	keys := sortedKeys(data)
	wrote := false
	writeLine := func(line string) error {
		if wrote {
			if err := buffer.WriteByte('\n'); err != nil {
				return err
			}
		}
		wrote = true
		_, err := buffer.WriteString(line)
		return err
	}
	for _, key := range keys {
		if _, isSection := data[key].(map[string]any); isSection {
			continue
		}
		line, err := iniLine(key, data[key])
		if err != nil {
			return nil, err
		}
		if err := writeLine(line); err != nil {
			return nil, err
		}
	}
	for _, key := range keys {
		section, isSection := data[key].(map[string]any)
		if !isSection {
			continue
		}
		if err := writeLine("[" + key + "]"); err != nil {
			return nil, err
		}
		for _, sectionKey := range sortedKeys(section) {
			if _, nested := section[sectionKey].(map[string]any); nested {
				return nil, fmt.Errorf("%w: [%s].%s", ErrData, key, sectionKey)
			}
			line, err := iniLine(sectionKey, section[sectionKey])
			if err != nil {
				return nil, fmt.Errorf("[%s]: %w", key, err)
			}
			if err := writeLine(line); err != nil {
				return nil, err
			}
		}
	}
	return buffer.Bytes(), nil
}

func iniLine(key string, value any) (string, error) {
	text, err := formatScalar(value)
	if err != nil {
		return "", fmt.Errorf("%s: %w", key, err)
	}
	return key + " = " + text, nil
}

func formatScalar(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case bool:
		return strconv.FormatBool(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("%w: %T", ErrData, value)
	}
}

func encodeXML(data map[string]any) ([]byte, error) {
	keys := sortedKeys(data)
	if len(keys) != 1 {
		return nil, fmt.Errorf("%w: got %d top-level keys", ErrData, len(keys))
	}
	root := keys[0]
	if _, isList := data[root].([]any); isList {
		return nil, fmt.Errorf("%w: root %q is a list", ErrData, root)
	}
	var buffer bytes.Buffer
	if _, err := buffer.WriteString(xml.Header); err != nil {
		return nil, err
	}
	if err := (xmlWriter{buffer: &buffer}).node(root, data[root], 0); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type xmlWriter struct {
	buffer *bytes.Buffer
}

func (writer xmlWriter) node(name string, value any, indent int) error {
	if err := checkXMLName(name); err != nil {
		return err
	}
	switch typed := value.(type) {
	case []any:
		for index, element := range typed {
			if _, nested := element.([]any); nested {
				return fmt.Errorf("%s: %w: nested list", name, ErrData)
			}
			if index > 0 {
				if err := writer.buffer.WriteByte('\n'); err != nil {
					return err
				}
			}
			if err := writer.node(name, element, indent); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		if err := writer.indent(indent); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer.buffer, "<%s>", name); err != nil {
			return err
		}
		children := sortedKeys(typed)
		if len(children) == 0 {
			_, err := fmt.Fprintf(writer.buffer, "</%s>", name)
			return err
		}
		if err := writer.buffer.WriteByte('\n'); err != nil {
			return err
		}
		for index, child := range children {
			if index > 0 {
				if err := writer.buffer.WriteByte('\n'); err != nil {
					return err
				}
			}
			if err := writer.node(child, typed[child], indent+1); err != nil {
				return err
			}
		}
		if err := writer.buffer.WriteByte('\n'); err != nil {
			return err
		}
		if err := writer.indent(indent); err != nil {
			return err
		}
		_, err := fmt.Fprintf(writer.buffer, "</%s>", name)
		return err
	default:
		text, err := formatScalar(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := writer.indent(indent); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(writer.buffer, "<%s>", name); err != nil {
			return err
		}
		if err := xml.EscapeText(writer.buffer, []byte(text)); err != nil {
			return err
		}
		_, err = fmt.Fprintf(writer.buffer, "</%s>", name)
		return err
	}
}

func (writer xmlWriter) indent(level int) error {
	for range level {
		if _, err := writer.buffer.WriteString("  "); err != nil {
			return err
		}
	}
	return nil
}

func checkXMLName(name string) error {
	if name == "" || !validXMLName(name) {
		return fmt.Errorf("%w: %q", ErrData, name)
	}
	return nil
}

func validXMLName(name string) bool {
	char, size := utf8.DecodeRuneInString(name)
	if size == 0 || char == utf8.RuneError || !isXMLNameStart(char) {
		return false
	}
	for _, next := range name[size:] {
		if !isXMLNameChar(next) {
			return false
		}
	}
	return true
}

func isXMLNameStart(char rune) bool {
	return char == '_' || unicode.IsLetter(char)
}

func isXMLNameChar(char rune) bool {
	return char == '_' || char == '-' || char == '.' || unicode.IsLetter(char) || unicode.IsDigit(char)
}
