package table

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"
)

// Write prints each value from seq in format.
// A nil layout uses Formatter on T when T has Columns, and otherwise the
// exported struct fields. A json name is the header, and json:"-" skips the
// field. A field that implements fmt.Stringer or error uses that text in
// table and CSV. JSONL keeps the field's JSON value unless Column.Format is set.
// A non-struct value is one column named value. Table and CSV print a header
// even when seq is empty. JSONL prints one object per line and no header.
func Write[T any](w io.Writer, format Format, seq iter.Seq[T], layout Formatter[T]) error {
	if err := format.validate(); err != nil {
		return err
	}
	chosen, err := Resolve(layout)
	if err != nil {
		return err
	}
	fields, err := bind(chosen)
	if err != nil {
		return err
	}
	if seq == nil {
		seq = func(func(T) bool) {}
	}
	switch format {
	case JSONL:
		return writeJSONL(w, seq, fields)
	case CSV:
		return writeCSV(w, seq, fields)
	default:
		return writeTable(w, seq, fields)
	}
}

func writeJSONL[T any](w io.Writer, seq iter.Seq[T], cols []col[T]) error {
	for value := range seq {
		line, err := jsonLine(value, cols)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(w, line); err != nil {
			return err
		}
	}
	return nil
}

func jsonLine[T any](value T, cols []col[T]) (string, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, c := range cols {
		if i > 0 {
			b.WriteByte(',')
		}
		name, err := json.Marshal(c.name)
		if err != nil {
			return "", err
		}
		cell, err := c.json(value)
		if err != nil {
			return "", err
		}
		raw, err := json.Marshal(cell)
		if err != nil {
			return "", err
		}
		b.Write(name)
		b.WriteByte(':')
		b.Write(raw)
	}
	b.WriteString("}\n")
	return b.String(), nil
}

func writeCSV[T any](w io.Writer, seq iter.Seq[T], cols []col[T]) error {
	out := csv.NewWriter(w)
	if err := out.Write(names(cols)); err != nil {
		return err
	}
	for value := range seq {
		row, err := cells(value, cols)
		if err != nil {
			return err
		}
		if err := out.Write(row); err != nil {
			return err
		}
	}
	out.Flush()
	return out.Error()
}

func writeTable[T any](w io.Writer, seq iter.Seq[T], cols []col[T]) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if err := writeLine(tw, names(cols)); err != nil {
		return err
	}
	for value := range seq {
		row, err := cells(value, cols)
		if err != nil {
			return err
		}
		if err := writeLine(tw, row); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeLine(w io.Writer, fields []string) error {
	_, err := io.WriteString(w, strings.Join(fields, "\t")+"\n")
	return err
}

func names[T any](cols []col[T]) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.name
	}
	return out
}

func cells[T any](value T, cols []col[T]) ([]string, error) {
	out := make([]string, len(cols))
	for i, c := range cols {
		text, err := c.text(value)
		if err != nil {
			return nil, err
		}
		out[i] = text
	}
	return out, nil
}

func jsonAt(v reflect.Value, index []int) any {
	v = fieldValue(v, index)
	if !v.IsValid() {
		return nil
	}
	if (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) && v.IsNil() {
		return nil
	}
	return v.Interface()
}

func cellAt(v reflect.Value, index []int) string {
	return formatValue(fieldValue(v, index))
}

func fieldValue(v reflect.Value, index []int) reflect.Value {
	if len(index) == 0 {
		return v
	}
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	return v.FieldByIndex(index)
}

func formatAny(v any) string {
	if v == nil {
		return ""
	}
	return formatValue(reflect.ValueOf(v))
}

func formatValue(v reflect.Value) string {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return ""
		}
		if v.Kind() == reflect.Interface {
			if err, ok := v.Interface().(error); ok {
				return err.Error()
			}
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return ""
	}
	if v.CanInterface() {
		if err, ok := v.Interface().(error); ok {
			return err.Error()
		}
		if text, ok := v.Interface().(fmt.Stringer); ok {
			return text.String()
		}
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	default:
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return fmt.Sprint(v.Interface())
		}
		return string(b)
	}
}
