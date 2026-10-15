package text

// Digits reports whether s is non-empty and every rune is ASCII 0-9.
func Digits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
