package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLockSpecUsesPin(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"dependencies":[{"kind":"tool","ref":"github:getsops/sops","currentValue":"v3.10.2"}]}`)
	if err := os.WriteFile(filepath.Join(dir, "modot.lock.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := LockSpec(dir, "github:getsops/sops")
	if err != nil {
		t.Fatal(err)
	}
	if spec != "github:getsops/sops@v3.10.2" {
		t.Fatalf("spec %q", spec)
	}
}

func TestLockSpecWalksToParent(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"dependencies":[{"kind":"tool","ref":"github:protocolbuffers/protobuf","currentValue":"v36.2"}]}`)
	if err := os.WriteFile(filepath.Join(dir, "modot.lock.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "nested")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	spec, err := LockSpec(child, "github:protocolbuffers/protobuf")
	if err != nil {
		t.Fatal(err)
	}
	if spec != "github:protocolbuffers/protobuf@v36.2" {
		t.Fatalf("spec %q", spec)
	}
}

func TestLockSpecLatestWithoutPin(t *testing.T) {
	spec, err := LockSpec(t.TempDir(), "github:getsops/sops")
	if err != nil {
		t.Fatal(err)
	}
	if spec != "github:getsops/sops@latest" {
		t.Fatalf("spec %q", spec)
	}
}

func TestLockSpecBadJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "modot.lock.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LockSpec(dir, "github:getsops/sops")
	if err == nil || !strings.HasPrefix(err.Error(), "modot.lock.json:") {
		t.Fatalf("err %v", err)
	}
}
