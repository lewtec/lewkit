package sops

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
)

func dataKey(recipients []ageRecipient) ([]byte, error) {
	ids, err := identities()
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: set SOPS_AGE_KEY, SOPS_AGE_KEY_FILE, or sops/age/keys.txt", ErrNotAge)
	}
	var last error
	for _, rec := range recipients {
		reader, err := age.Decrypt(armor.NewReader(strings.NewReader(rec.Enc)), ids...)
		if err != nil {
			last = err
			continue
		}
		plain, err := io.ReadAll(reader)
		if err != nil {
			last = err
			continue
		}
		return plain, nil
	}
	if last == nil {
		return nil, ErrNotAge
	}
	return nil, fmt.Errorf("%w: %v", ErrNotAge, last)
}

func identities() ([]age.Identity, error) {
	if strings.TrimSpace(os.Getenv("SOPS_AGE_KEY_CMD")) != "" {
		return nil, fmt.Errorf("sops: SOPS_AGE_KEY_CMD is not supported")
	}
	var out []age.Identity
	if text := os.Getenv("SOPS_AGE_KEY"); strings.TrimSpace(text) != "" {
		ids, err := parseIdentities(strings.NewReader(text))
		if err != nil {
			return nil, fmt.Errorf("sops: SOPS_AGE_KEY: %w", err)
		}
		out = append(out, ids...)
	}
	if path := strings.TrimSpace(os.Getenv("SOPS_AGE_KEY_FILE")); path != "" {
		ids, err := identitiesFrom(path)
		if err != nil {
			return nil, fmt.Errorf("sops: SOPS_AGE_KEY_FILE: %w", err)
		}
		out = append(out, ids...)
	}
	dir, err := userConfigDir()
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("sops: %w", err)
	}
	if dir != "" {
		path := filepath.Join(dir, "sops", "age", "keys.txt")
		ids, err := identitiesFrom(path)
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("sops: %w", err)
		}
		out = append(out, ids...)
	}
	return out, nil
}

func identitiesFrom(path string) ([]age.Identity, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseIdentities(bytes.NewReader(raw))
}

func parseIdentities(r io.Reader) ([]age.Identity, error) {
	var out []age.Identity
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		id, err := age.ParseX25519Identity(line)
		if err != nil {
			return nil, fmt.Errorf("invalid age identity")
		}
		out = append(out, id)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func userConfigDir() (string, error) {
	if runtime.GOOS == "darwin" {
		if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
			return dir, nil
		}
	}
	return os.UserConfigDir()
}
