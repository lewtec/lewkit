package uikit

import (
	"testing"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/test"
)

func TestFactoryRejectsNonIOS(t *testing.T) {
	test.IncompatibleOff(t, factory{}.CheckCompatibility, build.GOOSIOS)
}
