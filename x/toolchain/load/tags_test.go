package load

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitTags(t *testing.T) {
	assert.Nil(t, SplitTags(""))
	assert.Nil(t, SplitTags(" \t"))
	assert.Equal(t, []string{"appmode"}, SplitTags("appmode"))
	assert.Equal(t, []string{"a", "b", "c"}, SplitTags("a,b c"))
	assert.Equal(t, []string{"a", "b"}, SplitTags(" a,,b\t"))
}
