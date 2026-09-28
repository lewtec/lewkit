package release

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// appID is the reverse-domain identity of this binary.
// A release stamp sets it with -X github.com/lewtec/lewkit/x/release.appID=br.tec.lew.myapp.
// An empty stamp falls back to LEWKIT_APP_ID.
var appID string

var reverseDomainPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

var (
	// ErrAppIDRequired means the stamp and LEWKIT_APP_ID are both empty.
	ErrAppIDRequired = errors.New("app id is required (reverse-domain, e.g. br.tec.lew.myapp)")
	// ErrAppIDTooLong means the id is longer than 200 bytes.
	ErrAppIDTooLong = errors.New("app id too long")
	// ErrAppIDPathChars means the id contains a path separator or "..".
	ErrAppIDPathChars = errors.New("app id must not contain path separators")
	// ErrAppIDNotReverseDNS means the id is not reverse-domain notation.
	ErrAppIDNotReverseDNS = errors.New("app id must be reverse-domain notation (e.g. br.tec.lew.myapp)")
)

// RequireStamp panics unless this binary was stamped with an app id and a version.
// A standalone go run leaves both empty (version defaults to dev).
func RequireStamp() (id, versionName string) {
	id = strings.TrimSpace(appID)
	versionName = strings.TrimSpace(version)
	if id == "" || versionName == "" || versionName == "dev" {
		panic("x/release stamp is missing: this binary is not a release build")
	}
	if err := ValidateAppID(id); err != nil {
		panic(err)
	}
	return id, versionName
}

// AppID returns the reverse-domain id stamped into this binary.
func AppID() (string, error) {
	id := strings.TrimSpace(appID)
	if id == "" {
		id = strings.TrimSpace(os.Getenv("LEWKIT_APP_ID"))
	}
	if err := ValidateAppID(id); err != nil {
		return "", err
	}
	return id, nil
}

// ValidateAppID checks a reverse-domain application id.
func ValidateAppID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrAppIDRequired
	}
	if len(id) > 200 {
		return ErrAppIDTooLong
	}
	if strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return ErrAppIDPathChars
	}
	if !reverseDomainPattern.MatchString(id) {
		return fmt.Errorf("%w: %q", ErrAppIDNotReverseDNS, id)
	}
	return nil
}
