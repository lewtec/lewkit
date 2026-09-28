package apk

import "testing"

func TestNDKTriple(t *testing.T) {
	got, err := ndkTriple("arm64")
	if err != nil || got != "aarch64-linux-android" {
		t.Fatalf("arm64: %s %v", got, err)
	}
	got, err = ndkTriple("arm")
	if err != nil || got != "armv7a-linux-androideabi" {
		t.Fatalf("arm: %s %v", got, err)
	}
}
