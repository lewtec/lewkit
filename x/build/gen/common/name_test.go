package common

import (
	"path/filepath"
	"testing"
)

func TestProductName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		id, name, want string
	}{
		{"br.tec.lew.counter", "Counter", "Counter"},
		{"br.tec.lew.counter", "My App", "My-App"},
		{"br.tec.lew.counter", "", "counter"},
		{"br.tec.lew.x", "!!!", "x"},
	}
	for _, tt := range tests {
		t.Run(tt.want+"/"+tt.name, func(t *testing.T) {
			t.Parallel()
			got := ProductName(tt.id, tt.name)
			if got != tt.want {
				t.Fatalf("ProductName(%q, %q) = %q; want %q", tt.id, tt.name, got, tt.want)
			}
		})
	}
}

func TestDefaultOutApp(t *testing.T) {
	t.Parallel()
	got := DefaultOutApp("Counter", "/tmp/proj")
	want := filepath.Join("/tmp/proj", "dist", "Counter.app")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	got = DefaultOutApp("Counter.app", "/tmp/proj")
	if got != want {
		t.Fatalf("suffix: got %q want %q", got, want)
	}
}

func TestXMLEscape(t *testing.T) {
	t.Parallel()
	got := XMLEscape(`A & B <C> "d"`)
	want := `A &amp; B &lt;C&gt; &quot;d&quot;`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
