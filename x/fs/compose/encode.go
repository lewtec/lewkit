package compose

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	iofs "io/fs"
	"maps"
	"slices"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

// Encode writes the combined bytes of one file.
// base opens [Ref] slots. A nil base rejects those slots.
func Encode(file File, base iofs.FS) ([]byte, error) {
	normalized, err := file.normalized()
	if err != nil {
		return nil, err
	}
	if err := normalized.check(); err != nil {
		return nil, err
	}
	switch normalized.Type {
	case TypeLines, TypeText, TypeRef:
		return encodeSlots(normalized, base)
	case TypeJSON, TypeTOML, TypeYAML, TypeINI, TypeXML:
		data, err := encodeStructured(normalized.Type, normalized.Data)
		if err != nil {
			return nil, err
		}
		return data, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrType, normalized.Type)
	}
}

func encodeSlots(file File, base iofs.FS) ([]byte, error) {
	keys := slices.Collect(maps.Keys(file.Values))
	slices.Sort(keys)
	var buffer bytes.Buffer
	for index, key := range keys {
		if index > 0 {
			if err := buffer.WriteByte('\n'); err != nil {
				return nil, err
			}
		}
		if err := file.Values[key].writeTo(&buffer, base); err != nil {
			return nil, fmt.Errorf("values.%s: %w", key, err)
		}
	}
	return buffer.Bytes(), nil
}

func (slot Slot) writeTo(writer io.Writer, base iofs.FS) error {
	switch slot.kind {
	case slotText:
		_, err := io.WriteString(writer, slot.body)
		return err
	case slotRef:
		if base == nil {
			return fmt.Errorf("%w: nil base", ErrRef)
		}
		opened, err := base.Open(slot.body)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrRef, err)
		}
		_, copyErr := io.Copy(writer, opened)
		closeErr := opened.Close()
		if copyErr != nil {
			return fmt.Errorf("%w: %w", ErrRef, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("%w: %w", ErrRef, closeErr)
		}
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrSlot, slot.kind)
	}
}

func encodeStructured(fileType Type, data map[string]any) ([]byte, error) {
	if data == nil {
		data = map[string]any{}
	}
	var (
		out []byte
		err error
	)
	switch fileType {
	case TypeJSON:
		out, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, err
		}
		out = append(out, '\n')
	case TypeTOML:
		out, err = toml.Marshal(data)
	case TypeYAML:
		out, err = encodeYAML(data)
	case TypeINI:
		out, err = encodeINI(data)
	case TypeXML:
		out, err = encodeXML(data)
	default:
		return nil, fmt.Errorf("%w: %q", ErrType, fileType)
	}
	if err != nil {
		return nil, err
	}
	return ensureNewline(out), nil
}

func ensureNewline(body []byte) []byte {
	if len(body) == 0 || body[len(body)-1] == '\n' {
		return body
	}
	return append(body, '\n')
}

func sortedKeys(data map[string]any) []string {
	keys := slices.Collect(maps.Keys(data))
	slices.Sort(keys)
	return keys
}

func encodeYAML(data map[string]any) ([]byte, error) {
	node, err := yamlNode(data)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(node); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func yamlNode(value any) (*yaml.Node, error) {
	switch typed := value.(type) {
	case map[string]any:
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		for _, key := range sortedKeys(typed) {
			node.Content = append(node.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key})
			child, err := yamlNode(typed[key])
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child)
		}
		return node, nil
	case []any:
		node := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		for _, element := range typed {
			child, err := yamlNode(element)
			if err != nil {
				return nil, err
			}
			node.Content = append(node.Content, child)
		}
		return node, nil
	case string:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: typed}, nil
	case bool:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(typed)}, nil
	case int64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatInt(typed, 10)}, nil
	case float64:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!float", Value: strconv.FormatFloat(typed, 'g', -1, 64)}, nil
	case nil:
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"}, nil
	default:
		return nil, fmt.Errorf("%w: %T", ErrData, value)
	}
}

func encodeINI(data map[string]any) ([]byte, error) {
	var buffer bytes.Buffer
	keys := sortedKeys(data)
	wrote := false
	for _, key := range keys {
		if _, isSection := data[key].(map[string]any); isSection {
			continue
		}
		line, err := iniLine(key, data[key])
		if err != nil {
			return nil, err
		}
		if wrote {
			if err := buffer.WriteByte('\n'); err != nil {
				return nil, err
			}
		}
		wrote = true
		if _, err := buffer.WriteString(line); err != nil {
			return nil, err
		}
	}
	for _, key := range keys {
		section, isSection := data[key].(map[string]any)
		if !isSection {
			continue
		}
		if wrote {
			if err := buffer.WriteByte('\n'); err != nil {
				return nil, err
			}
		}
		wrote = true
		if _, err := fmt.Fprintf(&buffer, "[%s]", key); err != nil {
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
			if _, err := buffer.WriteString("\n" + line); err != nil {
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
	encoder := xmlEncoder{buffer: &buffer}
	if err := encoder.node(root, data[root], 0); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

type xmlEncoder struct {
	buffer *bytes.Buffer
}

func (encoder xmlEncoder) node(name string, value any, indent int) error {
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
				if err := encoder.buffer.WriteByte('\n'); err != nil {
					return err
				}
			}
			if err := encoder.node(name, element, indent); err != nil {
				return err
			}
		}
		return nil
	case map[string]any:
		if err := encoder.indent(indent); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(encoder.buffer, "<%s>", name); err != nil {
			return err
		}
		children := sortedKeys(typed)
		if len(children) == 0 {
			_, err := fmt.Fprintf(encoder.buffer, "</%s>", name)
			return err
		}
		if err := encoder.buffer.WriteByte('\n'); err != nil {
			return err
		}
		for index, child := range children {
			if index > 0 {
				if err := encoder.buffer.WriteByte('\n'); err != nil {
					return err
				}
			}
			if err := encoder.node(child, typed[child], indent+1); err != nil {
				return err
			}
		}
		if err := encoder.buffer.WriteByte('\n'); err != nil {
			return err
		}
		if err := encoder.indent(indent); err != nil {
			return err
		}
		_, err := fmt.Fprintf(encoder.buffer, "</%s>", name)
		return err
	default:
		text, err := formatScalar(value)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		if err := encoder.indent(indent); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(encoder.buffer, "<%s>", name); err != nil {
			return err
		}
		if err := xml.EscapeText(encoder.buffer, []byte(text)); err != nil {
			return err
		}
		_, err = fmt.Fprintf(encoder.buffer, "</%s>", name)
		return err
	}
}

func (encoder xmlEncoder) indent(level int) error {
	for range level {
		if _, err := encoder.buffer.WriteString("  "); err != nil {
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
	rune, size := utf8.DecodeRuneInString(name)
	if size == 0 || rune == utf8.RuneError || !isXMLNameStart(rune) {
		return false
	}
	for _, char := range name[size:] {
		if !isXMLNameChar(char) {
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
