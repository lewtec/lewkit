package sops

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	sopsv3 "github.com/getsops/sops/v3"
	sopsdecrypt "github.com/getsops/sops/v3/decrypt"
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

// Parse reads path. A SOPS file decrypts. Any other file is kept as stored.
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
	if format == "" {
		format = sniff(in)
	}
	if !encrypted(in, format) {
		return in, nil
	}
	out, err := sopsdecrypt.Data(in, format)
	if errors.Is(err, sopsv3.MetadataNotFound) {
		return in, nil
	}
	if err != nil {
		if format == "ini" {
			err = fmt.Errorf("sops: ini: %w", err)
		}
		return nil, wrapDecrypt(err)
	}
	return out, nil
}

func encrypted(in []byte, format string) bool {
	switch format {
	case "":
		return false
	case "dotenv":
		return looksLikeDotenv(in)
	case "ini":
		return looksLikeINI(in)
	case "yaml":
		return bytes.Contains(in, []byte("sops:"))
	case "json", "binary":
		return bytes.Contains(in, []byte(`"sops"`))
	default:
		return true
	}
}

func sniff(in []byte) string {
	trim := bytes.TrimSpace(in)
	if len(trim) > 0 && trim[0] == '{' {
		return "binary"
	}
	if !textFile(in) {
		return ""
	}
	if looksLikeDotenv(in) {
		return "dotenv"
	}
	if looksLikeINI(in) {
		return "ini"
	}
	if bytes.Contains(in, []byte("sops:")) {
		return "yaml"
	}
	return ""
}

func wrapDecrypt(err error) error {
	msg := err.Error()
	if strings.Contains(msg, "mac") || strings.Contains(msg, "MAC") || strings.Contains(msg, "integrity") {
		return fmt.Errorf("%w: %w", ErrMAC, err)
	}
	return fmt.Errorf("%w: %w", ErrNotAge, err)
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
