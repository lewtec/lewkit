package compression

import (
	"bytes"
	"strings"
)

// HasName reports whether any item is named name.
func HasName[T interface{ Name() string }](items []T, name string) bool {
	for _, item := range items {
		if item.Name() == name {
			return true
		}
	}
	return false
}

// LongestExtension returns the item whose suffix of name is longest.
// An empty suffix is skipped. A suffix without a leading dot is dotted.
// No match is the zero value and false.
func LongestExtension[T interface{ Extensions() []string }](items []T, name string) (T, bool) {
	name = strings.ToLower(name)
	var best T
	bestN := -1
	for _, item := range items {
		for _, ext := range item.Extensions() {
			ext = strings.ToLower(ext)
			if ext == "" {
				continue
			}
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			if strings.HasSuffix(name, ext) && len(ext) > bestN {
				best = item
				bestN = len(ext)
			}
		}
	}
	if bestN < 0 {
		var zero T
		return zero, false
	}
	return best, true
}

// LongestMagic returns the item whose magic is the longest prefix of p.
// An empty prefix is skipped. No match is the zero value and false.
func LongestMagic[T interface{ Magic() [][]byte }](items []T, p []byte) (T, bool) {
	var best T
	bestN := -1
	for _, item := range items {
		for _, mag := range item.Magic() {
			if len(mag) == 0 || len(mag) <= bestN || len(p) < len(mag) {
				continue
			}
			if bytes.HasPrefix(p, mag) {
				best = item
				bestN = len(mag)
			}
		}
	}
	if bestN < 0 {
		var zero T
		return zero, false
	}
	return best, true
}

// Match returns the longest extension of name, or else the longest magic prefix of p.
func Match[T interface {
	Extensions() []string
	Magic() [][]byte
}](items []T, name string, p []byte) (T, bool) {
	if item, ok := LongestExtension(items, name); ok {
		return item, true
	}
	return LongestMagic(items, p)
}
