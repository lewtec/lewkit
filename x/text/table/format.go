// Package table writes a sequence of values as a table, JSONL, or CSV.
package table

import "fmt"

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum

// Format selects how Write prints a sequence. The zero value is invalid.
//
//lewkit:enum
type Format int

const (
	Table Format = iota + 1
	JSONL
	CSV
)

func (format Format) validate() error {
	if !format.IsValid() {
		return fmt.Errorf("%w: %d", ErrFormat, int(format))
	}
	return nil
}
