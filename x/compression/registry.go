package compression

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// ErrNil is [Register] with a nil codec.
var ErrNil = errors.New("nil codec")

// ErrExist is [Register] when the name is already in the registry.
var ErrExist = errors.New("already registered")

var std = New()

// Register adds one codec to the process-wide registry.
func Register(c Codec) error {
	if c == nil {
		return ErrNil
	}
	if hasName(std.codecs, c.Name()) {
		return fmt.Errorf("%s: %w", c.Name(), ErrExist)
	}
	std.codecs = append(std.codecs, c)
	return nil
}

// MustRegister is [Register] that panics on error.
func MustRegister(c Codec) {
	if err := Register(c); err != nil {
		panic(err)
	}
}

func hasName(codecs []Codec, name string) bool {
	for _, c := range codecs {
		if c.Name() == name {
			return true
		}
	}
	return false
}

// Detect looks up a codec in the process-wide registry.
// A matching extension wins over magic.
func Detect(name string, magic []byte) (Codec, bool) {
	return std.Detect(name, magic)
}

// ByExtension looks up a codec in the process-wide registry.
func ByExtension(name string) (Codec, bool) {
	return std.ByExtension(name)
}

// ByMagic looks up a codec in the process-wide registry.
func ByMagic(p []byte) (Codec, bool) {
	return std.ByMagic(p)
}

// Registry looks up codecs by extension or magic.
type Registry struct {
	codecs []Codec
}

// New builds a registry from codecs. The slice is copied.
func New(codecs ...Codec) *Registry {
	return &Registry{codecs: append([]Codec(nil), codecs...)}
}

// Detect returns a codec for name or magic.
// A matching extension wins over magic.
func (r *Registry) Detect(name string, magic []byte) (Codec, bool) {
	if c, ok := r.ByExtension(name); ok {
		return c, true
	}
	return r.ByMagic(magic)
}

// ByExtension returns the codec with the longest matching suffix of name.
func (r *Registry) ByExtension(name string) (Codec, bool) {
	name = strings.ToLower(name)
	var best Codec
	bestN := -1
	for _, c := range r.codecs {
		for _, ext := range c.Extensions() {
			ext = strings.ToLower(ext)
			if ext == "" {
				continue
			}
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			if strings.HasSuffix(name, ext) && len(ext) > bestN {
				best = c
				bestN = len(ext)
			}
		}
	}
	if bestN < 0 {
		return nil, false
	}
	return best, true
}

// ByMagic returns the codec with the longest matching magic prefix of p.
func (r *Registry) ByMagic(p []byte) (Codec, bool) {
	var best Codec
	bestN := -1
	for _, c := range r.codecs {
		for _, mag := range c.Magic() {
			if len(mag) == 0 || len(mag) <= bestN || len(p) < len(mag) {
				continue
			}
			if bytes.HasPrefix(p, mag) {
				best = c
				bestN = len(mag)
			}
		}
	}
	if bestN < 0 {
		return nil, false
	}
	return best, true
}
