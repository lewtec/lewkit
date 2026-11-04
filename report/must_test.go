package report

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustActuallyWorksAsExpected(t *testing.T) {
	boringFun := func() (string, error) {
		return "", os.ErrClosed
	}
	assert.Panics(t, func() {
		_ = Must(boringFun)
	})
}
