package window

import "os"
import "testing"

func TestMain(m *testing.M) {
	_ = os.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	os.Exit(m.Run())
}
