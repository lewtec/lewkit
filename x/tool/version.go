package tool

import (
	"strconv"
	"strings"
)

type semanticVersion struct {
	original string
	parts    []int
}

func parseVersion(version string) semanticVersion {
	original := version
	if version == "latest" {
		return semanticVersion{original: version}
	}
	version = strings.TrimPrefix(version, "v")
	rawParts := strings.Split(version, ".")
	parts := make([]int, 0, len(rawParts))
	for _, part := range rawParts {
		parts = append(parts, numericPrefix(part))
	}
	return semanticVersion{original: original, parts: parts}
}

func numericPrefix(part string) int {
	digits := ""
	for _, r := range part {
		if r < '0' || r > '9' {
			break
		}
		digits += string(r)
	}
	if digits == "" {
		return 0
	}
	number, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}
	return number
}

func (version semanticVersion) compare(other semanticVersion) int {
	if version.original == "latest" && other.original != "latest" {
		return 1
	}
	if version.original != "latest" && other.original == "latest" {
		return -1
	}
	if version.original == "latest" && other.original == "latest" {
		return 0
	}
	length := len(version.parts)
	if len(other.parts) > length {
		length = len(other.parts)
	}
	for i := 0; i < length; i++ {
		left := 0
		right := 0
		if i < len(version.parts) {
			left = version.parts[i]
		}
		if i < len(other.parts) {
			right = other.parts[i]
		}
		if left < right {
			return -1
		}
		if left > right {
			return 1
		}
	}
	return 0
}

// CompareVersions orders two version strings.
// It returns -1 when left is older, 0 when they match, and 1 when left is newer.
// The literal latest sorts after every concrete version.
func CompareVersions(left, right string) int {
	return parseVersion(left).compare(parseVersion(right))
}

func normalizeVersion(version string) string {
	version = strings.TrimPrefix(version, "v")
	return strings.ReplaceAll(version, "/", "-")
}
