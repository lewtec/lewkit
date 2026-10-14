package errs

import "strings"

// Classify reports the meta sentinel that name and text belong to.
// collapse is true when text is that meta condition and the caller should
// use the meta sentinel itself. parent is the Err name in this package,
// or empty when text is not one of the known conditions.
func Classify(name, text string) (parent string, collapse bool) {
	text = strings.TrimSpace(text)
	if text == "" || !strings.HasPrefix(name, "Err") {
		return "", false
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "nil context") || strings.Contains(name, "NilContext") {
		return "ErrNilContext", true
	}
	if lower == "syntax" || lower == "cycle" || packagePrefixed(lower, "syntax") || packagePrefixed(lower, "cycle") {
		if lower == "syntax" || packagePrefixed(lower, "syntax") {
			return "ErrSyntax", true
		}
		return "ErrCycle", true
	}
	if lower == "canceled" || lower == "cancelled" || packagePrefixed(lower, "canceled") || packagePrefixed(lower, "cancelled") {
		return "ErrCanceled", true
	}
	parent = match(name, lower)
	if parent == "" {
		return "", false
	}
	if lower == phrase(parent) {
		return parent, true
	}
	return parent, false
}

// packagePrefixed reports text like "tui: canceled", a single word, a colon, and the phrase.
func packagePrefixed(lower, phrase string) bool {
	prefix, rest, ok := strings.Cut(lower, ": ")
	if !ok || strings.TrimSpace(rest) != phrase {
		return false
	}
	return prefix != "" && !strings.Contains(prefix, " ")
}

func phrase(parent string) string {
	switch parent {
	case "ErrNotFound":
		return "not found"
	case "ErrInvalid":
		return "invalid"
	case "ErrMissing":
		return "missing"
	case "ErrUnsupported":
		return "unsupported"
	case "ErrAlreadyExists":
		return "already exists"
	case "ErrUnavailable":
		return "unavailable"
	case "ErrClosed":
		return "closed"
	case "ErrCanceled":
		return "canceled"
	case "ErrEmpty":
		return "empty"
	case "ErrNilContext":
		return "nil context"
	case "ErrSyntax":
		return "syntax"
	case "ErrCycle":
		return "cycle"
	case "ErrDuplicate":
		return "duplicate"
	case "ErrOutOfRange":
		return "out of range"
	case "ErrRequired":
		return "required"
	case "ErrUnknown":
		return "unknown"
	default:
		return ""
	}
}

func match(name, lower string) string {
	switch {
	case strings.Contains(lower, "not found") || strings.Contains(lower, "no such") || strings.Contains(name, "NotFound") || strings.Contains(name, "NoSuch"):
		return "ErrNotFound"
	case strings.Contains(lower, "unsupported") || strings.Contains(lower, "not supported") || strings.Contains(lower, "not a supported") || strings.Contains(name, "Unsupported"):
		return "ErrUnsupported"
	case strings.Contains(lower, "unavailable") || strings.Contains(lower, "not available") || strings.Contains(lower, "no available") || strings.Contains(name, "Unavailable"):
		return "ErrUnavailable"
	case strings.Contains(lower, "already exist") || strings.Contains(lower, "already registered") || strings.Contains(lower, "already open") || strings.Contains(lower, "already set") || name == "ErrExist" || name == "ErrRegistered" || strings.HasSuffix(name, "Registered"):
		return "ErrAlreadyExists"
	case strings.Contains(lower, "not empty") || strings.Contains(name, "NotEmpty"):
		return "ErrAlreadyExists"
	case strings.Contains(lower, "missing") || strings.Contains(name, "Missing"):
		return "ErrMissing"
	case strings.Contains(lower, "invalid") || strings.Contains(name, "Invalid") || strings.HasPrefix(lower, "bad ") || strings.Contains(lower, " bad ") || strings.Contains(lower, "is not a ") || strings.Contains(lower, "is not an ") || strings.Contains(lower, "not a "):
		return "ErrInvalid"
	case lower == "closed" || strings.HasSuffix(lower, " closed") || name == "ErrClosed" || strings.HasSuffix(name, "Closed"):
		return "ErrClosed"
	case strings.Contains(lower, "canceled") || strings.Contains(lower, "cancelled") || name == "ErrCanceled":
		return "ErrCanceled"
	case strings.Contains(lower, "no items") || strings.Contains(lower, "empty") || name == "ErrEmpty" || (strings.Contains(name, "Empty") && !strings.Contains(name, "NotEmpty")):
		return "ErrEmpty"
	case strings.Contains(lower, "duplicate") || strings.Contains(name, "Duplicate"):
		return "ErrDuplicate"
	case strings.Contains(lower, "out of range"):
		return "ErrOutOfRange"
	case strings.Contains(lower, "required") || strings.Contains(name, "Required"):
		return "ErrRequired"
	case strings.Contains(lower, "unknown") || strings.Contains(name, "Unknown"):
		return "ErrUnknown"
	case strings.Contains(lower, "has no ") || strings.HasPrefix(lower, "no ") || strings.Contains(lower, " no "):
		return "ErrNotFound"
	case strings.Contains(lower, "cycle"):
		return "ErrCycle"
	default:
		return ""
	}
}
