package metal

import (
	"testing"

	"github.com/lewtec/lewkit/x/test"
)

func TestFactoryRejectsNonApple(t *testing.T) {
	test.IncompatibleOff(t, factory{}.CheckCompatibility, "darwin", "ios")
}
