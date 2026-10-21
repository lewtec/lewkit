//go:build !android

package progress

import "github.com/mattn/go-runewidth"

func cellWidth(s string) int {
	return runewidth.StringWidth(s)
}

func clipCells(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if cellWidth(s) <= max {
		return s
	}
	return runewidth.Truncate(s, max, "")
}
