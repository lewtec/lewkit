package uikit

import (
	"testing"

	"github.com/lewtec/lewkit/x/test"
)

func TestFactoryRejectsNonIOS(t *testing.T) {
	test.IncompatibleOff(t, factory{}.CheckCompatibility, "ios")
}
