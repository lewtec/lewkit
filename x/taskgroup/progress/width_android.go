//go:build android

package progress

import "unicode/utf8"

func cellWidth(s string) int {
	return utf8.RuneCountInString(s)
}

func clipCells(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if cellWidth(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) > max {
		runes = runes[:max]
	}
	return string(runes)
}
