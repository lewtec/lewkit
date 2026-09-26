package common

import (
	"path/filepath"
	"strings"
	"unicode"
)

// ProductName is a filesystem-safe Xcode PRODUCT_NAME / .app stem.
func ProductName(packageID, appName string) string {
	name := strings.TrimSpace(appName)
	name = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			return r
		case r == ' ':
			return '-'
		default:
			return -1
		}
	}, name)
	name = strings.Trim(name, "-_")
	if name == "" {
		parts := strings.Split(packageID, ".")
		name = parts[len(parts)-1]
	}
	if name == "" {
		name = "App"
	}
	return name
}

// DefaultOutApp is dist/<app_name>.app under cwd.
func DefaultOutApp(appName, cwd string) string {
	name := strings.TrimSpace(appName)
	name = strings.TrimSuffix(name, ".app")
	if name == "" {
		name = "App"
	}
	return filepath.Join(cwd, "dist", name+".app")
}
