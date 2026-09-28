package build

import "testing"

func TestAndroidABI(t *testing.T) {
	abi, err := AndroidABI("arm64")
	if err != nil || abi != "arm64-v8a" {
		t.Fatalf("arm64: %s %v", abi, err)
	}
	if _, err := AndroidABI("ppc64"); err == nil {
		t.Fatal("expected error")
	}
}
