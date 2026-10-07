package sops

import (
	"bytes"
	"fmt"
	"strings"
)

// Env is a list of dotenv assignments. Later keys replace earlier ones.
type Env struct {
	keys []string
	vals map[string]string
}

// ParseEnv reads dotenv text. Empty lines and lines starting with # are
// skipped. A value's \n escape becomes a newline. Quotes stay in the value.
func ParseEnv(in []byte) (Env, error) {
	env := Env{vals: map[string]string{}}
	for _, line := range bytes.Split(in, []byte("\n")) {
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		pos := bytes.IndexByte(line, '=')
		if pos < 0 {
			return Env{}, fmt.Errorf("sops: invalid dotenv line")
		}
		key := string(line[:pos])
		if key == "" {
			return Env{}, fmt.Errorf("sops: empty dotenv key")
		}
		env.set(key, strings.ReplaceAll(string(line[pos+1:]), `\n`, "\n"))
	}
	return env, nil
}

// LoadEnv opens path and parses the plaintext as dotenv.
func LoadEnv(path string) (Env, error) {
	raw, err := Open(path)
	if err != nil {
		return Env{}, err
	}
	return ParseEnv(raw)
}

// Get returns the value for key.
func (e Env) Get(key string) (string, bool) {
	if e.vals == nil {
		return "", false
	}
	v, ok := e.vals[key]
	return v, ok
}

// Merge returns e with other's assignments applied after it.
func (e Env) Merge(other Env) Env {
	out := e.clone()
	for _, key := range other.keys {
		out.set(key, other.vals[key])
	}
	return out
}

// Strings returns KEY=VALUE lines. A repeated key appears once, with the later value.
func (e Env) Strings() []string {
	out := make([]string, len(e.keys))
	for i, key := range e.keys {
		out[i] = key + "=" + e.vals[key]
	}
	return out
}

func (e Env) clone() Env {
	out := Env{
		keys: append([]string(nil), e.keys...),
		vals: make(map[string]string, len(e.vals)),
	}
	for key, value := range e.vals {
		out.vals[key] = value
	}
	return out
}

func (e *Env) set(key, value string) {
	if e.vals == nil {
		e.vals = map[string]string{}
	}
	if _, ok := e.vals[key]; !ok {
		e.keys = append(e.keys, key)
	}
	e.vals[key] = value
}
