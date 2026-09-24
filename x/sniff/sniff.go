// Package sniff picks the longest file suffix or magic prefix.
package sniff

import (
	"bytes"
	"strings"
)

// ByExtension returns the entry with the longest matching suffix of name.
// A matching extension wins by length. Equal lengths keep the earlier entry.
func ByExtension[T interface{ Extensions() []string }](list []T, name string) (T, bool) {
	name = strings.ToLower(name)
	var best T
	bestN := -1
	for _, item := range list {
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

// ByMagic returns the entry with the longest matching magic prefix of p.
// Equal lengths keep the earlier entry.
func ByMagic[T interface{ Magic() [][]byte }](list []T, p []byte) (T, bool) {
	var best T
	bestN := -1
	for _, item := range list {
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

// HasName reports whether list already contains name.
func HasName[T interface{ Name() string }](list []T, name string) bool {
	for _, item := range list {
		if item.Name() == name {
			return true
		}
	}
	return false
}
