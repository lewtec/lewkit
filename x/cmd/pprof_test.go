package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPprofArgParse(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		directory string
		address   string
	}{
		{name: "empty", in: ""},
		{name: "absolute dir", in: "/tmp/p", directory: "/tmp/p"},
		{name: "relative dir", in: "./out", directory: "./out"},
		{name: "name", in: "profiles", directory: "profiles"},
		{name: "all interfaces", in: ":6060", address: ":6060"},
		{name: "localhost", in: "localhost:6060", address: "localhost:6060"},
		{name: "ipv4", in: "127.0.0.1:6060", address: "127.0.0.1:6060"},
		{name: "bare port", in: "8080", address: ":8080"},
		{name: "ipv6", in: "[::1]:443", address: "[::1]:443"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p PprofArg
			require.NoError(t, p.Parse(tc.in))
			assert.Equal(t, tc.in, p.Value())
			assert.Equal(t, tc.directory, p.Directory())
			assert.Equal(t, tc.address, p.Address())
		})
	}
}
