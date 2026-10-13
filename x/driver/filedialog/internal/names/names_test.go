package names

import "testing"

func TestTake(t *testing.T) {
	seen := map[string]struct{}{}
	if got := Take("a/b.txt", seen); got != "a_b.txt" {
		t.Fatal(got)
	}
	if got := Take("a/b.txt", seen); got != "a_b (2).txt" {
		t.Fatal(got)
	}
	if got := Take(".", seen); got != "_" {
		t.Fatal(got)
	}
	if got := Clean(`a\b`); got != "a_b" {
		t.Fatal(got)
	}
}
