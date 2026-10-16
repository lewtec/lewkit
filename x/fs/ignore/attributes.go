package ignore

import (
	"bufio"
	"bytes"
	"strconv"
	"strings"
)

// attributeKinds are .gitattributes marks that become ignore rules.
// The order is fixed so a line with both marks emits a stable sequence.
var attributeKinds = []string{
	"linguist-generated",
	"refactree-ignored",
}

// attributePatterns returns gitignore lines for linguist-generated and
// refactree-ignored marks. Other attributes are left out.
func attributePatterns(data []byte) []byte {
	var out bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "[") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pattern := unquotePattern(fields[0])
		if pattern == "" {
			continue
		}
		for _, kind := range attributeKinds {
			set, ok := attributeValue(fields[1:], kind)
			if !ok {
				continue
			}
			if !set {
				out.WriteByte('!')
			}
			out.WriteString(pattern)
			out.WriteByte('\n')
		}
	}
	return out.Bytes()
}

func unquotePattern(field string) string {
	if len(field) >= 2 && field[0] == '"' && field[len(field)-1] == '"' {
		if unquoted, err := strconv.Unquote(field); err == nil {
			return unquoted
		}
	}
	return field
}

// attributeValue reads a boolean git attribute.
// A bare name is set. -name, !name, and name=false are unset. The last
// mention of the name wins.
func attributeValue(attrs []string, want string) (set bool, ok bool) {
	for _, attr := range attrs {
		name, value, hasValue := splitAttribute(attr)
		if name != want {
			continue
		}
		ok = true
		if !hasValue {
			set = true
			continue
		}
		switch strings.ToLower(value) {
		case "true", "yes", "1", "set":
			set = true
		case "false", "no", "0", "unset":
			set = false
		default:
			set = value != ""
		}
	}
	return set, ok
}

func splitAttribute(attr string) (name, value string, hasValue bool) {
	if attr == "" {
		return "", "", false
	}
	switch attr[0] {
	case '-', '!':
		return attr[1:], "false", true
	}
	if i := strings.IndexByte(attr, '='); i >= 0 {
		return attr[:i], attr[i+1:], true
	}
	return attr, "", false
}
