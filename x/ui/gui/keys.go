package gui

import "github.com/lewtec/lewkit/x/driver/window"

func enterKey(key window.Key) bool {
	return key.Rune == '\r' || key.Rune == '\n' || key.Code == 36 || key.Code == 13 || key.Code == 24
}
