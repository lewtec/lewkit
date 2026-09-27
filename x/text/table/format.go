// Package table writes a sequence of values as a table, JSONL, or CSV.
package table

import (
	"errors"
	"fmt"
)

// ErrFormat means the format token is not one of table, jsonl, or csv.
var ErrFormat = errors.New("unknown format")

// Format selects how Write prints a sequence. The zero value is invalid.
type Format int

const (
	Table Format = iota + 1
	JSONL
	CSV
)

func (format Format) String() string {
	switch format {
	case Table:
		return "table"
	case JSONL:
		return "jsonl"
	case CSV:
		return "csv"
	default:
		return ""
	}
}

// Values lists the CLI tokens in usage order.
func (Format) Values() []Format {
	return []Format{Table, JSONL, CSV}
}

func (format Format) validate() error {
	if format.String() == "" {
		return fmt.Errorf("%w: %d", ErrFormat, int(format))
	}
	return nil
}
