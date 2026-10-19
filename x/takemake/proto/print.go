package proto

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Make writes the file as proto text.
func (file *File) Make(w io.Writer) error {
	if file == nil {
		return fmt.Errorf("%w: file", ErrValue)
	}
	if w == nil {
		return fmt.Errorf("%w: writer", ErrValue)
	}
	var buf bytes.Buffer
	if err := writeFile(&buf, file); err != nil {
		return err
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("%w: %w", ErrValue, err)
	}
	return nil
}

func writeFile(buf *bytes.Buffer, file *File) error {
	if file.Syntax != "" {
		fmt.Fprintf(buf, "syntax = %s;\n", strconv.Quote(file.Syntax))
	}
	if file.Package != "" {
		breakBefore(buf)
		fmt.Fprintf(buf, "package %s;\n", file.Package)
	}
	if len(file.Imports) > 0 {
		breakBefore(buf)
		for _, imported := range file.Imports {
			buf.WriteString("import ")
			if imported.Kind.IsValid() {
				buf.WriteString(imported.Kind.String())
				buf.WriteByte(' ')
			}
			buf.WriteString(strconv.Quote(imported.Path))
			buf.WriteString(";\n")
		}
	}
	if len(file.Options) > 0 {
		breakBefore(buf)
		for _, option := range file.Options {
			if err := writeOption(buf, "", option); err != nil {
				return err
			}
		}
	}
	for _, enum := range file.Enums {
		breakBefore(buf)
		if err := writeEnum(buf, 0, enum); err != nil {
			return err
		}
	}
	for _, message := range file.Messages {
		breakBefore(buf)
		if err := writeMessage(buf, 0, message); err != nil {
			return err
		}
	}
	for _, service := range file.Services {
		breakBefore(buf)
		if err := writeService(buf, service); err != nil {
			return err
		}
	}
	return nil
}

func breakBefore(buf *bytes.Buffer) {
	if buf.Len() == 0 || bytes.HasSuffix(buf.Bytes(), []byte("\n\n")) {
		return
	}
	buf.WriteByte('\n')
}

func writeMessage(buf *bytes.Buffer, level int, message *Message) error {
	if message == nil {
		return fmt.Errorf("%w: message", ErrValue)
	}
	indent := strings.Repeat("  ", level)
	fmt.Fprintf(buf, "%smessage %s {\n", indent, message.Name)
	inner := indent + "  "
	for _, option := range message.Options {
		if err := writeOption(buf, inner, option); err != nil {
			return err
		}
	}
	for _, reserved := range message.Reserved {
		if err := writeReserve(buf, inner, reserved); err != nil {
			return err
		}
	}
	for _, enum := range message.Enums {
		if err := writeEnum(buf, level+1, enum); err != nil {
			return err
		}
	}
	for _, nested := range message.Messages {
		if err := writeMessage(buf, level+1, nested); err != nil {
			return err
		}
	}
	for _, field := range message.Fields {
		if err := writeField(buf, inner, field); err != nil {
			return err
		}
	}
	for _, oneof := range message.Oneofs {
		if err := writeOneof(buf, level+1, oneof); err != nil {
			return err
		}
	}
	fmt.Fprintf(buf, "%s}\n", indent)
	return nil
}

func writeEnum(buf *bytes.Buffer, level int, enum *Enum) error {
	if enum == nil {
		return fmt.Errorf("%w: enum", ErrValue)
	}
	indent := strings.Repeat("  ", level)
	fmt.Fprintf(buf, "%senum %s {\n", indent, enum.Name)
	inner := indent + "  "
	for _, option := range enum.Options {
		if err := writeOption(buf, inner, option); err != nil {
			return err
		}
	}
	for _, reserved := range enum.Reserved {
		if err := writeReserve(buf, inner, reserved); err != nil {
			return err
		}
	}
	for _, value := range enum.Values {
		if value == nil {
			return fmt.Errorf("%w: enum value", ErrValue)
		}
		fmt.Fprintf(buf, "%s%s = %d", inner, value.Name, value.Number)
		if err := writeBrackets(buf, value.Options); err != nil {
			return err
		}
		buf.WriteString(";\n")
	}
	fmt.Fprintf(buf, "%s}\n", indent)
	return nil
}

func writeOneof(buf *bytes.Buffer, level int, oneof *Oneof) error {
	if oneof == nil {
		return fmt.Errorf("%w: oneof", ErrValue)
	}
	indent := strings.Repeat("  ", level)
	fmt.Fprintf(buf, "%soneof %s {\n", indent, oneof.Name)
	inner := indent + "  "
	for _, option := range oneof.Options {
		if err := writeOption(buf, inner, option); err != nil {
			return err
		}
	}
	for _, field := range oneof.Fields {
		if err := writeField(buf, inner, field); err != nil {
			return err
		}
	}
	fmt.Fprintf(buf, "%s}\n", indent)
	return nil
}

func writeField(buf *bytes.Buffer, indent string, field *Field) error {
	if field == nil {
		return fmt.Errorf("%w: field", ErrValue)
	}
	buf.WriteString(indent)
	if field.Label.IsValid() {
		buf.WriteString(field.Label.String())
		buf.WriteByte(' ')
	}
	buf.WriteString(field.Type.String())
	buf.WriteByte(' ')
	buf.WriteString(field.Name)
	buf.WriteString(" = ")
	buf.WriteString(strconv.Itoa(field.Number))
	if err := writeBrackets(buf, field.Options); err != nil {
		return err
	}
	buf.WriteString(";\n")
	return nil
}

func writeService(buf *bytes.Buffer, service *Service) error {
	if service == nil {
		return fmt.Errorf("%w: service", ErrValue)
	}
	fmt.Fprintf(buf, "service %s {\n", service.Name)
	for _, option := range service.Options {
		if err := writeOption(buf, "  ", option); err != nil {
			return err
		}
	}
	for _, method := range service.Methods {
		if err := writeMethod(buf, method); err != nil {
			return err
		}
	}
	buf.WriteString("}\n")
	return nil
}

func writeMethod(buf *bytes.Buffer, method *Method) error {
	if method == nil {
		return fmt.Errorf("%w: rpc", ErrValue)
	}
	buf.WriteString("  rpc ")
	buf.WriteString(method.Name)
	buf.WriteString(" (")
	if method.ClientStream {
		buf.WriteString("stream ")
	}
	buf.WriteString(method.Input)
	buf.WriteString(") returns (")
	if method.ServerStream {
		buf.WriteString("stream ")
	}
	buf.WriteString(method.Output)
	buf.WriteByte(')')
	if len(method.Options) == 0 {
		buf.WriteString(";\n")
		return nil
	}
	buf.WriteString(" {\n")
	for _, option := range method.Options {
		if err := writeOption(buf, "    ", option); err != nil {
			return err
		}
	}
	buf.WriteString("  }\n")
	return nil
}

func writeReserve(buf *bytes.Buffer, indent string, reserved Reserve) error {
	if len(reserved.Names) > 0 && len(reserved.Ranges) > 0 {
		return fmt.Errorf("%w: reserved", ErrValue)
	}
	if len(reserved.Names) == 0 && len(reserved.Ranges) == 0 {
		return fmt.Errorf("%w: reserved", ErrValue)
	}
	buf.WriteString(indent)
	buf.WriteString("reserved ")
	if len(reserved.Names) > 0 {
		for i, name := range reserved.Names {
			if i > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(strconv.Quote(name))
		}
		buf.WriteString(";\n")
		return nil
	}
	for i, rng := range reserved.Ranges {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(strconv.Itoa(rng.Start))
		if rng.Max {
			buf.WriteString(" to max")
			continue
		}
		if rng.End != rng.Start {
			buf.WriteString(" to ")
			buf.WriteString(strconv.Itoa(rng.End))
		}
	}
	buf.WriteString(";\n")
	return nil
}

func writeOption(buf *bytes.Buffer, indent string, option Option) error {
	buf.WriteString(indent)
	buf.WriteString("option ")
	buf.WriteString(option.Name)
	buf.WriteString(" = ")
	if err := writeValue(buf, option.Value, indent); err != nil {
		return err
	}
	buf.WriteString(";\n")
	return nil
}

func writeBrackets(buf *bytes.Buffer, options []Option) error {
	if len(options) == 0 {
		return nil
	}
	buf.WriteString(" [")
	for i, option := range options {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(option.Name)
		buf.WriteString(" = ")
		if err := writeValue(buf, option.Value, ""); err != nil {
			return err
		}
	}
	buf.WriteByte(']')
	return nil
}

func writeValue(buf *bytes.Buffer, value Value, indent string) error {
	switch value.Kind {
	case KindString:
		buf.WriteString(strconv.Quote(value.Text))
	case KindBool:
		if value.Bool {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case KindInt:
		buf.WriteString(strconv.FormatInt(value.Int, 10))
	case KindFloat:
		writeFloat(buf, value.Float)
	case KindIdent:
		if value.Text == "" {
			return fmt.Errorf("%w: identifier", ErrValue)
		}
		buf.WriteString(value.Text)
	case KindAggregate:
		if len(value.Entries) == 0 {
			buf.WriteString("{}")
			return nil
		}
		buf.WriteString("{\n")
		inner := indent + "  "
		for _, entry := range value.Entries {
			buf.WriteString(inner)
			buf.WriteString(entry.Name)
			buf.WriteString(": ")
			if err := writeValue(buf, entry.Value, inner); err != nil {
				return err
			}
			buf.WriteByte('\n')
		}
		buf.WriteString(indent)
		buf.WriteByte('}')
	case KindList:
		buf.WriteByte('[')
		for i, item := range value.List {
			if i > 0 {
				buf.WriteString(", ")
			}
			if err := writeValue(buf, item, indent); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		return fmt.Errorf("%w: constant", ErrValue)
	}
	return nil
}

func writeFloat(buf *bytes.Buffer, number float64) {
	switch {
	case math.IsNaN(number):
		buf.WriteString("nan")
	case math.IsInf(number, 1):
		buf.WriteString("inf")
	case math.IsInf(number, -1):
		buf.WriteString("-inf")
	default:
		text := strconv.FormatFloat(number, 'g', -1, 64)
		if !strings.ContainsAny(text, ".eE") {
			text += ".0"
		}
		buf.WriteString(text)
	}
}
