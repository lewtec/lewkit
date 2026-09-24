package cocoa

import (
	"net/url"
	"strings"
)

// dropPaths keeps local filesystem paths. file URLs are decoded. Other schemes are skipped.
func dropPaths(items []string) []string {
	var paths []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.HasPrefix(item, "file:") {
			parsed, err := url.Parse(item)
			if err != nil || parsed.Scheme != "file" {
				continue
			}
			path, err := url.PathUnescape(parsed.Path)
			if err != nil || path == "" {
				continue
			}
			paths = append(paths, path)
			continue
		}
		if strings.HasPrefix(item, "/") {
			paths = append(paths, item)
		}
	}
	return paths
}
