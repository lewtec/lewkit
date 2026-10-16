package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/test"
)

func TestCheckCompatibilitySkipsOtherSystems(t *testing.T) {
	test.IncompatibleOff(t, factory{}.CheckCompatibility, build.GOOSAndroid)
}
