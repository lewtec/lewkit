package dotenv

import (
	"errors"
	"fmt"
	"strings"
)

func parse(text string) (*File, error) {
	text = strings.TrimPrefix(text, "\uFEFF")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	file := New()
	var pending []string
	for n, line := range lines {
		if n == len(lines)-1 && line == "" {
			break
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			pending = append(pending, commentBody(trimmed))
			continue
		}
		assignment, err := parseAssignment(trimmed)
		if err != nil {
			return nil, lineError(err, n+1)
		}
		if len(pending) > 0 {
			assignment.Comment = joinComment(strings.Join(pending, "\n"), assignment.Comment)
			pending = nil
		}
		file.Assignments = append(file.Assignments, assignment)
	}
	if len(pending) == 0 {
		return file, nil
	}
	note := strings.Join(pending, "\n")
	if len(file.Assignments) == 0 {
		file.Assignments = append(file.Assignments, &Assignment{Comment: note})
		return file, nil
	}
	last := file.Assignments[len(file.Assignments)-1]
	last.Comment = joinComment(last.Comment, note)
	return file, nil
}

func lineError(err error, line int) error {
	if errors.Is(err, ErrKey) {
		return fmt.Errorf("%w: line %d", ErrKey, line)
	}
	return fmt.Errorf("%w: line %d", ErrLine, line)
}

func joinComment(lead, inline string) string {
	if lead == "" {
		return inline
	}
	if inline == "" {
		return lead
	}
	return lead + "\n" + inline
}

func parseAssignment(line string) (*Assignment, error) {
	assignment := &Assignment{Quote: QuoteBare}
	if rest, ok := strings.CutPrefix(line, "export"); ok && rest != "" && (rest[0] == ' ' || rest[0] == '\t') {
		assignment.Export = true
		line = strings.TrimLeft(rest, " \t")
	}
	key, rest, ok := splitKey(line)
	if !ok {
		return nil, ErrLine
	}
	if key == "" {
		return nil, ErrKey
	}
	rest = strings.TrimLeft(rest, " \t")
	if !strings.HasPrefix(rest, "=") {
		return nil, ErrLine
	}
	value, quote, comment, err := parseValue(rest[1:])
	if err != nil {
		return nil, err
	}
	assignment.Key = key
	assignment.Value = value
	assignment.Quote = quote
	assignment.Comment = comment
	return assignment, nil
}

func splitKey(s string) (key, rest string, ok bool) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '=' || c == ' ' || c == '\t' {
			return s[:i], s[i:], true
		}
		if !keyByte(c, i == 0) {
			return "", "", false
		}
	}
	return "", "", false
}

func keyByte(c byte, first bool) bool {
	if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' {
		return true
	}
	return !first && c >= '0' && c <= '9'
}

func parseValue(rest string) (value string, quote Quote, comment string, err error) {
	i := 0
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t') {
		i++
	}
	if i == len(rest) {
		return "", QuoteBare, "", nil
	}
	if rest[i] == '#' && i > 0 {
		return "", QuoteBare, commentBody(rest[i:]), nil
	}
	switch rest[i] {
	case '"':
		value, rest, err = scanDouble(rest[i+1:])
		if err != nil {
			return "", 0, "", err
		}
		comment, err = trailingComment(rest)
		return value, QuoteDouble, comment, err
	case '\'':
		value, rest, err = scanSingle(rest[i+1:])
		if err != nil {
			return "", 0, "", err
		}
		comment, err = trailingComment(rest)
		return value, QuoteSingle, comment, err
	default:
		value, comment = scanBare(rest[i:])
		return value, QuoteBare, comment, nil
	}
}

func scanDouble(s string) (value, rest string, err error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			return b.String(), s[i+1:], nil
		}
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		if i+1 >= len(s) {
			return "", "", ErrLine
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		default:
			b.WriteByte(s[i])
		}
	}
	return "", "", ErrLine
}

func scanSingle(s string) (value, rest string, err error) {
	i := strings.IndexByte(s, '\'')
	if i < 0 {
		return "", "", ErrLine
	}
	return s[:i], s[i+1:], nil
}

func scanBare(s string) (value, comment string) {
	for i := 0; i < len(s); i++ {
		if s[i] == '#' && i > 0 && (s[i-1] == ' ' || s[i-1] == '\t') {
			return strings.TrimRight(s[:i-1], " \t"), commentBody(s[i:])
		}
	}
	return strings.TrimRight(s, " \t"), ""
}

func trailingComment(rest string) (string, error) {
	i := 0
	for i < len(rest) && (rest[i] == ' ' || rest[i] == '\t') {
		i++
	}
	if i == len(rest) {
		return "", nil
	}
	if rest[i] != '#' {
		return "", ErrLine
	}
	return commentBody(rest[i:]), nil
}

func commentBody(s string) string {
	s = strings.TrimPrefix(s, "#")
	return strings.TrimPrefix(s, " ")
}
