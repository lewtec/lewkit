package tool

import (
	"path/filepath"
	"sort"
	"strings"
)

// ContainsAnyOf reports whether any needle is a substring of haystack.
func ContainsAnyOf(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(haystack, needle) {
			return true
		}
	}
	return false
}

// ScoreArtifact ranks how well artifact matches osName, arch, and binaryHint.
// Zero means ineligible. A positive score is eligible. Higher is better.
// Android accepts a linux build only as a fallback, and an exact OS match
// outranks that fallback.
func ScoreArtifact(artifact Artifact, osName, arch, binaryHint string) int {
	oses := []string{osName}
	if osName == "android" {
		oses = append(oses, "linux")
	}

	matchesPlatform := false
	for _, candidate := range oses {
		if artifact.OS == candidate && artifact.Arch == arch {
			matchesPlatform = true
			break
		}
	}
	if !matchesPlatform {
		return 0
	}
	if ContainsAnyOf(artifact.URL, ".deb", ".rpm") {
		return 0
	}

	hint := strings.ToLower(strings.TrimSpace(binaryHint))
	base := strings.ToLower(filepath.Base(artifact.URL))
	score := 0
	if artifact.OS == osName {
		score += 500
	}
	if hint != "" {
		for _, separator := range []string{"-", "_", "."} {
			if strings.Contains(base, hint+separator) || strings.Contains(base, separator+hint+separator) || strings.Contains(base, separator+hint+".") {
				score += 120
				break
			}
		}
		if strings.Contains(base, hint) {
			score += 60
		}
	} else {
		score += 1
	}
	if strings.HasSuffix(base, ".tar.gz") || strings.HasSuffix(base, ".tgz") ||
		strings.HasSuffix(base, ".zip") || strings.HasSuffix(base, ".tar.xz") {
		score += 10
	}
	if strings.Contains(base, "debug") {
		score -= 20
	}
	if ContainsAnyOf(base,
		"desktop", "appimage", ".dmg", "sbom", "provenance", "sigstore",
		"checksum", "sha256", "sha512", ".rpm", ".deb",
		"npm", "package", "app-server", "-zsh", "proxy", "symbols",
		"bundle", "sandbox", "command-runner", "responses-api", ".whl",
	) {
		score -= 50
	}
	if hint != "" {
		for _, archToken := range []string{"x86_64", "amd64", "aarch64", "arm64", "x64", "i686", "i386"} {
			if strings.Contains(base, hint+"-"+archToken) || strings.Contains(base, hint+"_"+archToken) {
				score += 40
				break
			}
		}
	}
	if score <= 0 {
		score = 1
	}
	return score
}

// SelectArtifact returns the highest-scoring eligible artifact, or nil.
// Equal scores prefer the shorter URL.
func SelectArtifact(artifacts []Artifact, osName, arch, binaryHint string) *Artifact {
	var candidates []Artifact
	for _, artifact := range artifacts {
		if ScoreArtifact(artifact, osName, arch, binaryHint) > 0 {
			candidates = append(candidates, artifact)
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := ScoreArtifact(candidates[i], osName, arch, binaryHint)
		right := ScoreArtifact(candidates[j], osName, arch, binaryHint)
		if left != right {
			return left > right
		}
		return len(candidates[i].URL) < len(candidates[j].URL)
	})
	return &candidates[0]
}
