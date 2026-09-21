package mise

import "testing"

func TestNewToolRejectsEmptyRef(t *testing.T) {
	if _, err := NewTool("  "); err == nil {
		t.Fatal("NewTool(empty) succeeded")
	}
}
