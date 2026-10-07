package sops

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// ErrNotAge means the file is SOPS but no age recipient could open it.
var ErrNotAge = errors.New("sops file requires an age recipient")

// ErrMAC means the decrypted values do not match the SOPS MAC.
var ErrMAC = errors.New("sops mac mismatch")

// File is a path to a secret file.
// An empty path leaves Value nil. After a successful Parse, Value is the
// plaintext, and an empty file is a non-nil empty slice.
type File struct {
	raw []byte
}

// Parse reads path. A SOPS age file decrypts. Any other file is kept as stored.
func (f *File) Parse(path string) error {
	if strings.TrimSpace(path) == "" {
		f.raw = nil
		return nil
	}
	raw, err := Open(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	f.raw = raw
	return nil
}

// Value returns the plaintext bytes. It is nil when Parse was not given a path.
func (f File) Value() []byte {
	return f.raw
}

// Text returns Value with trailing newlines removed, for a password file.
func (f File) Text() string {
	return strings.TrimRight(string(f.raw), "\r\n")
}

// Open reads path and returns its plaintext.
func Open(path string) ([]byte, error) {
	in, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return present(decode(in, formatOf(path)))
}

// Decode returns the plaintext of in.
// in is returned unchanged when it has no SOPS metadata.
// A JSON object with only data and sops is a SOPS binary file.
func Decode(in []byte) ([]byte, error) {
	return present(decode(in, ""))
}

func present(out []byte, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	if out == nil {
		return []byte{}, nil
	}
	return out, nil
}

func formatOf(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".env":
		return "dotenv"
	case ".ini":
		return "ini"
	default:
		return ""
	}
}

func decode(in []byte, format string) ([]byte, error) {
	switch format {
	case "yaml":
		return decodeYAML(in)
	case "json":
		return decodeJSON(in, false)
	case "dotenv":
		if looksLikeDotenv(in) {
			return nil, errors.New("sops: dotenv files are not supported")
		}
		return in, nil
	case "ini":
		if looksLikeINI(in) {
			return nil, errors.New("sops: ini files are not supported")
		}
		return in, nil
	default:
		if out, ok, err := tryJSON(in); ok || err != nil {
			return out, err
		}
		if out, ok, err := tryYAML(in); ok || err != nil {
			return out, err
		}
		if textFile(in) && looksLikeDotenv(in) {
			return nil, errors.New("sops: dotenv files are not supported")
		}
		if textFile(in) && looksLikeINI(in) {
			return nil, errors.New("sops: ini files are not supported")
		}
		return in, nil
	}
}

func textFile(in []byte) bool {
	return utf8.Valid(in) && !bytes.Contains(in, []byte{0})
}

func looksLikeDotenv(in []byte) bool {
	if !textFile(in) {
		return false
	}
	for _, line := range bytes.Split(in, []byte("\n")) {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("sops_mac=")) {
			return true
		}
	}
	return false
}

func looksLikeINI(in []byte) bool {
	if !textFile(in) {
		return false
	}
	for _, line := range bytes.Split(in, []byte("\n")) {
		if bytes.Equal(bytes.TrimSpace(line), []byte("[sops]")) {
			return true
		}
	}
	return false
}
