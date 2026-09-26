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

type column struct {
	name  string
	index []int
}

// Write prints each value from seq. Struct columns are the exported fields.
// A json name, when present, is the column name. json:"-" skips the field.
// A non-struct value is one column named value. Table and CSV print a header
// even when seq is empty. JSONL prints one object per line and no header.
func Write[T any](w io.Writer, format Format, seq iter.Seq[T]) error {
	if err := format.validate(); err != nil {
		return err
	}
	if seq == nil {
		seq = func(func(T) bool) {}
	}
	switch format {
	case JSONL:
		return writeJSONL(w, seq)
	case CSV:
		return writeCSV(w, seq)
	default:
		return writeTable(w, seq)
	}
}

func writeJSONL[T any](w io.Writer, seq iter.Seq[T]) error {
	enc := json.NewEncoder(w)
	for value := range seq {
		if err := enc.Encode(value); err != nil {
			return err
		}
	}
	return nil
}

func writeCSV[T any](w io.Writer, seq iter.Seq[T]) error {
	cols := columnsOf[T]()
	out := csv.NewWriter(w)
	if err := out.Write(names(cols)); err != nil {
		return err
	}
	for value := range seq {
		if err := out.Write(cells(reflect.ValueOf(value), cols)); err != nil {
			return err
		}
	}
	out.Flush()
	return out.Error()
}

func writeTable[T any](w io.Writer, seq iter.Seq[T]) error {
	cols := columnsOf[T]()
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if err := writeLine(tw, names(cols)); err != nil {
		return err
	}
	for value := range seq {
		if err := writeLine(tw, cells(reflect.ValueOf(value), cols)); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeLine(w io.Writer, fields []string) error {
	_, err := io.WriteString(w, strings.Join(fields, "\t")+"\n")
	return err
}

func names(cols []column) []string {
	out := make([]string, len(cols))
	for i, col := range cols {
		out[i] = col.name
	}
	return out
}

func columnsOf[T any]() []column {
	t := reflect.TypeFor[T]()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return []column{{name: "value"}}
	}
	var cols []column
	for _, field := range reflect.VisibleFields(t) {
		if field.Anonymous || !field.IsExported() {
			continue
		}
		name := field.Name
		if tag, ok := field.Tag.Lookup("json"); ok {
			name = tag
			if comma := strings.IndexByte(tag, ','); comma >= 0 {
				name = tag[:comma]
			}
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
		}
		cols = append(cols, column{name: name, index: field.Index})
	}
	return cols
}

func cells(v reflect.Value, cols []column) []string {
	out := make([]string, len(cols))
	for i, col := range cols {
		out[i] = cellAt(v, col.index)
	}
	return out
}

func cellAt(v reflect.Value, index []int) string {
	if len(index) == 0 {
		return formatValue(v)
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct || !v.IsValid() {
		return ""
	}
	return formatValue(v.FieldByIndex(index))
}

func formatValue(v reflect.Value) string {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if !v.IsValid() || v.IsNil() {
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
