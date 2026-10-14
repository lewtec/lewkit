package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/test"
)

func TestNotAndroid(t *testing.T) {
	test.IncompatibleOff(t, factory{}.CheckCompatibility, "android")
}
