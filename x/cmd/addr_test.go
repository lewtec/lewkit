package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddrArg(t *testing.T) {
	type args struct {
		addr AddrArg `long:"addr"`
	}
	cases := []struct {
		name string
		in   string
		want string
		host string
		port string
		err  error
	}{
		{name: "host port", in: "127.0.0.1:8080", want: "127.0.0.1:8080", host: "127.0.0.1", port: "8080"},
		{name: "all interfaces", in: ":8080", want: ":8080", host: "", port: "8080"},
		{name: "bare port", in: "8080", want: ":8080", host: "", port: "8080"},
		{name: "ipv6", in: "[::1]:443", want: "[::1]:443", host: "::1", port: "443"},
		{name: "hostname", in: "localhost:80", want: "localhost:80", host: "localhost", port: "80"},
		{name: "missing port", in: "localhost", err: ErrInvalidArgument},
		{name: "empty", in: "", err: ErrInvalidArgument},
		{name: "empty port", in: "localhost:", err: ErrInvalidArgument},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse[args]("--addr", tc.in)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.addr.Value())
			assert.Equal(t, tc.host, got.addr.Host())
			assert.Equal(t, tc.port, got.addr.Port())
		})
	}
}

func TestAddrDefault(t *testing.T) {
	type args struct {
		addr AddrArg `long:"addr" default:":8080"`
	}
	got, err := Parse[args]()
	require.NoError(t, err)
	assert.Equal(t, ":8080", got.addr.Value())
}
