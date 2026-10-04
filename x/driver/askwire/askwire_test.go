package askwire

import "testing"

func TestRoundTrip(t *testing.T) {
	raw := Format(StatusOK, "hello\nthere")
	status, payload := Split(raw)
	if status != StatusOK || payload != "hello\nthere" {
		t.Fatalf("status %q payload %q", status, payload)
	}
}

func TestSplitMissingNewline(t *testing.T) {
	status, payload := Split(StatusCanceled)
	if status != StatusCanceled || payload != "" {
		t.Fatalf("status %q payload %q", status, payload)
	}
}
