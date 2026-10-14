package android

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPercentOf(t *testing.T) {
	if percentOf(0.5, false) != 0 {
		require.FailNow(t, fmt.Sprint(percentOf(0.5, false)))
	}
	if percentOf(0.5, true) != 50 {
		require.FailNow(t, fmt.Sprint(percentOf(0.5, true)))
	}
	if percentOf(1, true) != 100 || percentOf(-1, true) != 0 || percentOf(2, true) != 100 {
		require.FailNow(t, fmt.Sprint(percentOf(1, true), percentOf(-1, true), percentOf(2, true)))
	}
}
