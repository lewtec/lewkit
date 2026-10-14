package tailwindcss

import (
	"testing"

	"github.com/lewtec/lewkit/x/test"
)

func TestLoad(t *testing.T) {
	test.RenderContains(t, Load().Render, Path)
}
