package android

import (
	"fmt"
	"strconv"
	"strings"
)

// A catalog is one record per line.
// A record is name, size, dir, and uri, separated by tabs.
// name and uri escape backslash, newline, carriage return, and tab.
// dir is 0 or 1. An empty catalog is an empty string.
func parseCatalog(s string) ([]Doc, error) {
	if s == "" {
		return nil, nil
	}
	s = strings.TrimSuffix(s, "\n")
	lines := strings.Split(s, "\n")
	out := make([]Doc, 0, len(lines))
	for _, line := range lines {
		doc, err := parseRecord(line)
		if err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, nil
}

func parseRecord(line string) (Doc, error) {
	parts := strings.Split(line, "\t")
	if len(parts) != 4 {
		return Doc{}, fmt.Errorf("%w: record", errCatalog)
	}
	name, err := unescape(parts[0])
	if err != nil {
		return Doc{}, err
	}
	size, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Doc{}, fmt.Errorf("%w: size", errCatalog)
	}
	if parts[2] != "0" && parts[2] != "1" {
		return Doc{}, fmt.Errorf("%w: dir", errCatalog)
	}
	uri, err := unescape(parts[3])
	if err != nil {
		return Doc{}, err
	}
	return Doc{Name: name, Size: size, Dir: parts[2] == "1", URI: uri}, nil
}

func unescape(s string) (string, error) {
	if !strings.Contains(s, "\\") {
		return s, nil
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 >= len(s) {
			return "", fmt.Errorf("%w: escape", errCatalog)
		}
		i++
		switch s[i] {
		case '\\':
			b.WriteByte('\\')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		default:
			return "", fmt.Errorf("%w: escape", errCatalog)
		}
	}
	return b.String(), nil
}
