package vulkan

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPresentExtent(t *testing.T) {
	w, h := presentExtent(2400, 1080, transformIdent)
	assert.Equal(t, 2400, w)
	assert.Equal(t, 1080, h)
	w, h = presentExtent(2400, 1080, transform90)
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
}

func TestPresentSourceRotatesUprightPixel(t *testing.T) {
	const width, height = 8, 4
	sx, sy := presentSource(transform90, width, height, 0, 0)
	assert.Equal(t, 0, sx)
	assert.Equal(t, height-1, sy)
	sx, sy = presentSource(transform270, width, height, 0, 0)
	assert.Equal(t, width-1, sx)
	assert.Equal(t, 0, sy)
	sx, sy = presentSource(transform180, width, height, 1, 2)
	assert.Equal(t, width-2, sx)
	assert.Equal(t, height-3, sy)
}
