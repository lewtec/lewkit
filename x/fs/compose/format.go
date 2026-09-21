package compose

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
)

func encodeStructured(fileType Type, data map[string]any) ([]byte, error) {
	if data == nil {
		data = map[string]any{}
	}
	var (
		body []byte
		err  error
	)
	switch fileType {
	case TypeJSON:
		body, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return nil, err
		}
		body = append(body, '\n')
	case TypeTOML:
		body, err = toml.Marshal(data)
	case TypeYAML:
		body, err = yaml.Marshal(data)
	case TypeINI:
		body, err = encodeINI(data)
	case TypeXML:
		body, err = encodeXML(data)
	default:
		return nil, fmt.Errorf("%w: %q", ErrType, fileType)
	}
	if err != nil {
		return nil, err
	}
	return ensureNewline(body), nil
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
