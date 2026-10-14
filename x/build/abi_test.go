package build

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAndroidABI(t *testing.T) {
	abi, err := AndroidABI("arm64")
	if err != nil || abi != "arm64-v8a" {
		require.FailNowf(t, "", "arm64: %s %v", abi, err)
	}
	if _, err := AndroidABI("ppc64"); err == nil {
		require.FailNow(t, fmt.Sprint("expected error"))
	}
}
