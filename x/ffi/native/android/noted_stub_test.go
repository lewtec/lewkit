//go:build !android

package android

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotedVMIsFalseOffAndroid(t *testing.T) {
	if NotedVM() {
		require.FailNow(t, fmt.Sprint("host build noted a Java VM"))
	}
}
