package prelude_test

import (
	"slices"
	"testing"

	"github.com/lewtec/lewkit/x/tool"
	_ "github.com/lewtec/lewkit/x/tool/prelude"
	"github.com/lewtec/lewkit/x/tool/registry"
)

func TestPreludeRegistersBackends(t *testing.T) {
	for _, id := range []string{"github", "mise", "registry"} {
		backend, err := tool.Get(id)
		if err != nil {
			t.Fatalf("Get(%q): %v", id, err)
		}
		if backend.Name() == "" {
			t.Fatalf("backend %q has an empty name", id)
		}
	}
	names := registry.ListTools()
	if !slices.Contains(names, "uv") {
		t.Fatalf("curated names = %v, want uv", names)
	}
	if !slices.Contains(names, "golang") {
		t.Fatalf("curated names = %v, want golang", names)
	}
}
