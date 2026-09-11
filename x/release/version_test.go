package release

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionVCS(t *testing.T) {
	cases := []struct {
		version string
		vcs     string
		got     string
	}{
		{
			version: "",
			vcs:     "aaaaaaaaa",
			got:     "dev-aaaaaaaaa",
		},
		{
			version: "",
			vcs:     "",
			got:     "dev",
		},
	}

	for _, item := range cases {
		assert.Equal(t, formatVersion(item.version, item.vcs), item.got, "should match")
	}
}

func TestPrintVersion(t *testing.T) {
	var buf bytes.Buffer
	assert.NoError(t, PrintVersion(&buf))
	assert.Equal(t, Version()+"\n", buf.String())
}
