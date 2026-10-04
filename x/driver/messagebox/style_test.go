package messagebox

import "testing"

func TestNormalizeStyle(t *testing.T) {
	cases := map[string]string{
		"":              StyleInformational,
		"info":          StyleInformational,
		"informational": StyleInformational,
		"warning":       StyleWarning,
		"warn":          StyleWarning,
		"critical":      StyleCritical,
		"error":         StyleCritical,
		"nope":          StyleInformational,
	}
	for in, want := range cases {
		if got := NormalizeStyle(in); got != want {
			t.Fatalf("NormalizeStyle(%q) = %q, want %q", in, got, want)
		}
	}
}
