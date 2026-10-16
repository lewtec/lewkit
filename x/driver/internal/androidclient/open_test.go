package androidclient

import (
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/test"
)

func TestOpenNotAndroid(t *testing.T) {
	test.IncompatibleOff(t, func(ctx context.Context) error {
		_, err := Open(ctx)
		return err
	}, build.GOOSAndroid)
}

func TestInteractiveNotAndroid(t *testing.T) {
	test.IncompatibleOff(t, Interactive, build.GOOSAndroid)
}
