package dotenv

import (
	"fmt"
	"io"
	"strings"

	syntax "github.com/lewtec/lewkit/x/parse/dotenv"
)

func writeAssignment(w io.Writer, assignment *Assignment) error {
	if assignment == nil {
		return ErrNilAssignment
	}
	if assignment.Key == "" {
		if assignment.Value != "" || assignment.Export || assignment.Quote != 0 {
			return ErrKey
		}
		return writeComment(w, assignment.Comment)
	}
	if err := validateKey(assignment.Key); err != nil {
		return err
	}
	if err := representable(assignment); err != nil {
		return err
	}
	if err := writeComment(w, assignment.Comment); err != nil {
		return err
	}
	if assignment.Export {
		if err := writeString(w, "export "); err != nil {
			return err
		}
	}
	if err := writeString(w, assignment.Key); err != nil {
		return err
	}
	if err := writeByte(w, '='); err != nil {
		return err
	}
	switch assignment.Quote {
	case QuoteDouble:
		if err := writeByte(w, '"'); err != nil {
			return err
		}
		if err := writeDouble(w, assignment.Value); err != nil {
			return err
		}
		if err := writeByte(w, '"'); err != nil {
			return err
		}
	case QuoteSingle:
		if err := writeByte(w, '\''); err != nil {
			return err
		}
		if err := writeString(w, assignment.Value); err != nil {
			return err
		}
		if err := writeByte(w, '\''); err != nil {
			return err
		}
	case QuoteBare:
		if err := writeString(w, assignment.Value); err != nil {
			return err
		}
	default:
		return ErrQuote
	}
	return writeByte(w, '\n')
}

func writeString(w io.Writer, text string) error {
	_, err := io.WriteString(w, text)
	return err
}

func writeByte(w io.Writer, c byte) error {
	var buf [1]byte
	buf[0] = c
	_, err := w.Write(buf[:])
	return err
}

func validateKey(key string) error {
	if key == "" {
		return ErrKey
	}
	for i := 0; i < len(key); i++ {
		if !syntax.KeyByte(key[i], i == 0) {
			return fmt.Errorf("%w: %s", ErrLine, key)
		}
	}
	return nil
}

func representable(assignment *Assignment) error {
	switch assignment.Quote {
	case QuoteDouble:
		return nil
	case QuoteSingle:
		if strings.Contains(assignment.Value, "'") || strings.ContainsAny(assignment.Value, "\n\r") {
			return ErrEncode
		}
		return nil
	case QuoteBare:
		if !bareOK(assignment.Value) {
			return ErrEncode
		}
		return nil
	default:
		return ErrQuote
	}
}

func bareOK(value string) bool {
	if strings.ContainsAny(value, "\n\r") {
		return false
	}
	if strings.HasPrefix(value, " ") || strings.HasPrefix(value, "\t") {
		return false
	}
	if strings.HasSuffix(value, " ") || strings.HasSuffix(value, "\t") {
		return false
	}
	if strings.Contains(value, " #") || strings.Contains(value, "\t#") {
		return false
	}
	return true
}

func writeComment(w io.Writer, comment string) error {
	if comment == "" {
		return nil
	}
	for line := range strings.SplitSeq(comment, "\n") {
		if err := writeByte(w, '#'); err != nil {
			return err
		}
		if line != "" {
			if err := writeByte(w, ' '); err != nil {
				return err
			}
			if err := writeString(w, line); err != nil {
				return err
			}
		}
		if err := writeByte(w, '\n'); err != nil {
			return err
		}
	}
	return nil
}

func writeDouble(w io.Writer, value string) error {
	start := 0
	for i := 0; i < len(value); i++ {
		var esc string
		switch value[i] {
		case '\\':
			esc = `\\`
		case '"':
			esc = `\"`
		case '\n':
			esc = `\n`
		case '\r':
			esc = `\r`
		case '\t':
			esc = `\t`
		default:
			continue
		}
		if err := writeRaw(w, value, start, i); err != nil {
			return err
		}
		if err := writeString(w, esc); err != nil {
			return err
		}
		start = i + 1
	}
	return writeRaw(w, value, start, len(value))
}

func writeRaw(w io.Writer, value string, start, end int) error {
	if start >= end {
		return nil
	}
	return writeString(w, value[start:end])
}
