package dotenv

import (
	"fmt"
	"strings"
)

func writeAssignment(b *strings.Builder, assignment *Assignment) error {
	if assignment == nil {
		return ErrNilAssignment
	}
	if assignment.Key == "" {
		if assignment.Value != "" || assignment.Export || assignment.Quote != 0 {
			return ErrKey
		}
		writeComment(b, assignment.Comment)
		return nil
	}
	if err := validateKey(assignment.Key); err != nil {
		return err
	}
	if err := representable(assignment); err != nil {
		return err
	}
	writeComment(b, assignment.Comment)
	if assignment.Export {
		b.WriteString("export ")
	}
	b.WriteString(assignment.Key)
	b.WriteByte('=')
	switch assignment.Quote {
	case QuoteDouble:
		b.WriteByte('"')
		b.WriteString(encodeDouble(assignment.Value))
		b.WriteByte('"')
	case QuoteSingle:
		b.WriteByte('\'')
		b.WriteString(assignment.Value)
		b.WriteByte('\'')
	case QuoteBare:
		b.WriteString(assignment.Value)
	default:
		return ErrQuote
	}
	b.WriteByte('\n')
	return nil
}

func validateKey(key string) error {
	if key == "" {
		return ErrKey
	}
	for i := 0; i < len(key); i++ {
		if !keyByte(key[i], i == 0) {
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

func writeComment(b *strings.Builder, comment string) {
	if comment == "" {
		return
	}
	for line := range strings.SplitSeq(comment, "\n") {
		b.WriteByte('#')
		if line != "" {
			b.WriteByte(' ')
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
}

func encodeDouble(value string) string {
	var b strings.Builder
	b.Grow(len(value))
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(value[i])
		}
	}
	return b.String()
}
