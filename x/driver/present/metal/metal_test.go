package metal

import (
	"testing"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/test"
)

func TestFactoryRejectsNonApple(t *testing.T) {
	test.IncompatibleOff(t, factory{}.CheckCompatibility, build.GOOSDarwin, build.GOOSIOS)
}
