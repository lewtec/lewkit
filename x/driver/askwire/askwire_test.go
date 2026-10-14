package askwire

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	raw := Format(StatusOK, "hello\nthere")
	status, payload := Split(raw)
	if status != StatusOK || payload != "hello\nthere" {
		require.FailNowf(t, "", "status %q payload %q", status, payload)
	}
}

func TestSplitMissingNewline(t *testing.T) {
	status, payload := Split(StatusCanceled)
	if status != StatusCanceled || payload != "" {
		require.FailNowf(t, "", "status %q payload %q", status, payload)
	}
}
