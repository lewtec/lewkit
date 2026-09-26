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

// Write prints each value from sequence in format.
// A nil layout uses Formatter on Row when Row has Columns, and otherwise the
// exported struct fields. A json name is the header, and json:"-" skips the
// field. A field that implements fmt.Stringer or error uses that text in
// table and CSV. JSONL keeps the field's JSON value unless Column.Format is set.
// A non-struct value is one column named value. Table and CSV print a header
// even when sequence is empty. JSONL prints one object per line and no header.
func Write[Row any](writer io.Writer, format Format, sequence iter.Seq[Row], layout Formatter[Row]) error {
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
	if sequence == nil {
		sequence = func(func(Row) bool) {}
	}
	switch format {
	case JSONL:
		return writeJSONL(writer, sequence, fields)
	case CSV:
		return writeCSV(writer, sequence, fields)
	default:
		return writeTable(writer, sequence, fields)
	}
}

func writeJSONL[Row any](writer io.Writer, sequence iter.Seq[Row], columns []boundColumn[Row]) error {
	for value := range sequence {
		line, err := jsonLine(value, columns)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(writer, line); err != nil {
			return err
		}
	}
	return nil
}

func jsonLine[Row any](value Row, columns []boundColumn[Row]) (string, error) {
	var builder strings.Builder
	builder.WriteByte('{')
	for index, column := range columns {
		if index > 0 {
			builder.WriteByte(',')
		}
		name, err := json.Marshal(column.name)
		if err != nil {
			return "", err
		}
		cell, err := column.json(value)
		if err != nil {
			return "", err
		}
		raw, err := json.Marshal(cell)
		if err != nil {
			return "", err
		}
		builder.Write(name)
		builder.WriteByte(':')
		builder.Write(raw)
	}
	builder.WriteString("}\n")
	return builder.String(), nil
}

func writeCSV[Row any](writer io.Writer, sequence iter.Seq[Row], columns []boundColumn[Row]) error {
	csvWriter := csv.NewWriter(writer)
	if err := csvWriter.Write(names(columns)); err != nil {
		return err
	}
	for value := range sequence {
		row, err := cells(value, columns)
		if err != nil {
			return err
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}
	csvWriter.Flush()
	return csvWriter.Error()
}

func writeTable[Row any](writer io.Writer, sequence iter.Seq[Row], columns []boundColumn[Row]) error {
	tabWriter := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	if err := writeLine(tabWriter, names(columns)); err != nil {
		return err
	}
	for value := range sequence {
		row, err := cells(value, columns)
		if err != nil {
			return err
		}
		if err := writeLine(tabWriter, row); err != nil {
			return err
		}
	}
	return tabWriter.Flush()
}

func writeLine(writer io.Writer, fields []string) error {
	_, err := io.WriteString(writer, strings.Join(fields, "\t")+"\n")
	return err
}

func names[Row any](columns []boundColumn[Row]) []string {
	out := make([]string, len(columns))
	for index, column := range columns {
		out[index] = column.name
	}
	return out
}

func cells[Row any](value Row, columns []boundColumn[Row]) ([]string, error) {
	out := make([]string, len(columns))
	for index, column := range columns {
		text, err := column.text(value)
		if err != nil {
			return nil, err
		}
		out[index] = text
	}
	return out, nil
}

func jsonAt(value reflect.Value, index []int) any {
	value = fieldValue(value, index)
	if !value.IsValid() {
		return nil
	}
	if (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) && value.IsNil() {
		return nil
	}
	return value.Interface()
}

func fieldValue(value reflect.Value, index []int) reflect.Value {
	if len(index) == 0 {
		return value
	}
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	return value.FieldByIndex(index)
}

func formatAny(value any) string {
	if value == nil {
		return ""
	}
	return formatValue(reflect.ValueOf(value))
}

func formatValue(value reflect.Value) string {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return ""
		}
		if value.Kind() == reflect.Interface {
			if err, ok := value.Interface().(error); ok {
				return err.Error()
			}
		}
		value = value.Elem()
	}
	if !value.IsValid() {
		return ""
	}
	if value.CanInterface() {
		if err, ok := value.Interface().(error); ok {
			return err.Error()
		}
		if text, ok := value.Interface().(fmt.Stringer); ok {
			return text.String()
		}
	}
	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Bool:
		return strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	default:
		encoded, err := json.Marshal(value.Interface())
		if err != nil {
			return fmt.Sprint(value.Interface())
		}
		return string(encoded)
	}
}
