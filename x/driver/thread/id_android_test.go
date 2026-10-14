//go:build android

package thread

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOSThreadStable(t *testing.T) {
	id := OSThread()
	if id == 0 {
		require.FailNow(t, fmt.Sprint("tid is 0"))
	}
	if again := OSThread(); again != id {
		require.FailNowf(t, "", "tid %d then %d", id, again)
	}
}
