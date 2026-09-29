package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/dirs"
	"github.com/stretchr/testify/require"
)

func TestPathsFromDataDir(t *testing.T) {
	got := pathsFromDataDir("/data/user/0/br.tec.lew.drivers")
	want := dirs.Dirs{
		Data:   "/data/user/0/br.tec.lew.drivers/files",
		Cache:  "/data/user/0/br.tec.lew.drivers/cache",
		Config: "/data/user/0/br.tec.lew.drivers/files/config",
		Inbox:  "/data/user/0/br.tec.lew.drivers/cache/inbox",
	}
	require.Equal(t, want, got)
}
