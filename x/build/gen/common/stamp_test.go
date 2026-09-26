package common

import (
	"testing"
)

func TestStampPackagingVersion_FillsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	vi, name, code := StampPackagingVersion(dir, "", 0)
	if vi.Version == "" {
		t.Fatal("empty version info")
	}
	if name == "" {
		t.Fatal("empty name")
	}
	if code <= 0 {
		t.Fatalf("code=%d", code)
	}
}

func TestStampPackagingVersion_KeepsExplicit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, name, code := StampPackagingVersion(dir, "9.8.7", 42)
	if name != "9.8.7" {
		t.Fatalf("name=%q", name)
	}
	if code != 42 {
		t.Fatalf("code=%d", code)
	}
}
