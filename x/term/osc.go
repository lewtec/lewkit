package term

import (
	"net/url"
	"strings"
)

const (
	oscLimit   = 70_000
	factKeyMax = 64
	factValMax = 1 << 16
	factBagMax = 64
)

// report is one OSC fact or directory from a pane's output.
type report struct {
	cwd string
	set bool
	del bool
	key string
	val string
}

func parseOSC(payload string) (report, bool) {
	switch {
	case strings.HasPrefix(payload, "7;"):
		path, ok := fileURLPath(payload[2:])
		if !ok {
			return report{}, false
		}
		return report{cwd: path}, true
	case strings.HasPrefix(payload, "1337;"):
		path, ok := oscField(payload[len("1337;"):], "CurrentDir")
		if !ok {
			return report{}, false
		}
		return report{cwd: path}, true
	case strings.HasPrefix(payload, "777;"):
		return parseFactOSC(payload[len("777;"):])
	default:
		return report{}, false
	}
}

func parseFactOSC(body string) (report, bool) {
	op, rest, ok := cutSemi(body)
	if !ok {
		return report{}, false
	}
	switch op {
	case "cwd":
		path := strings.TrimSpace(rest)
		if path == "" || strings.ContainsAny(path, "\x00\n\r") {
			return report{}, false
		}
		return report{cwd: path}, true
	case "del":
		key := strings.TrimSpace(rest)
		if !validFactKey(key) {
			return report{}, false
		}
		return report{del: true, key: key}, true
	case "set":
		key, val, ok := cutSemi(rest)
		if !ok || !validFactKey(key) {
			return report{}, false
		}
		val = trimFact(val)
		if strings.ContainsRune(val, 0) {
			return report{}, false
		}
		return report{set: true, key: key, val: val}, true
	default:
		return report{}, false
	}
}

func cutSemi(s string) (head, tail string, ok bool) {
	i := strings.IndexByte(s, ';')
	if i < 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}

func oscField(body, name string) (string, bool) {
	prefix := name + "="
	for _, part := range strings.Split(body, ";") {
		if strings.HasPrefix(part, prefix) {
			value := strings.TrimSpace(part[len(prefix):])
			if value == "" {
				return "", false
			}
			return value, true
		}
	}
	return "", false
}

func fileURLPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	if strings.HasPrefix(raw, "/") {
		return raw, true
	}
	if !strings.HasPrefix(raw, "file:") {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Path == "" {
		return "", false
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil || path == "" {
		return "", false
	}
	return path, true
}

func validFactKey(key string) bool {
	if key == "" || len(key) > factKeyMax {
		return false
	}
	for _, r := range key {
		if r <= 0x20 || r > 0x7e || r == ';' || r == '\\' {
			return false
		}
	}
	return true
}

func trimFact(value string) string {
	if len(value) > factValMax {
		return value[:factValMax]
	}
	return value
}
