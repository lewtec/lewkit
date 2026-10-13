package nativeerr

import (
	"errors"
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/driver/present"
)

func TestMap(t *testing.T) {
	lost := errors.New("lost")
	size := errors.New("size")
	closed := errors.New("closed")
	other := errors.New("other")
	if Map(nil, lost, size, closed) != nil {
		t.Fatal("nil")
	}
	if !errors.Is(Map(fmt.Errorf("%w", lost), lost, size, closed), present.ErrLost) {
		t.Fatal("lost")
	}
	if !errors.Is(Map(size, lost, size, closed), present.ErrSize) {
		t.Fatal("size")
	}
	if !errors.Is(Map(closed, lost, size, closed), present.ErrClosed) {
		t.Fatal("closed")
	}
	if !errors.Is(Map(other, lost, size, closed), other) {
		t.Fatal("other")
	}
}
