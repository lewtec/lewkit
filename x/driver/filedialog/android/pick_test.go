package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/stretchr/testify/require"
)

func TestFactoryIdentity(t *testing.T) {
	f := factory{}
	require.Equal(t, "filedialog_android", f.ID())
	require.Equal(t, "Android file chooser", f.Name())
	require.Equal(t, 80, f.Weight())
}

func TestDoctorListsChooser(t *testing.T) {
	var found bool
	for _, iface := range driver.Doctor(t.Context()) {
		for _, item := range iface.Drivers {
			if item.ID != "filedialog_android" {
				continue
			}
			found = true
			require.Equal(t, 80, item.Weight)
			require.Equal(t, "Android file chooser", item.Name)
		}
	}
	require.True(t, found)
}

func TestExtensionList(t *testing.T) {
	got := extensionList([]filedialog.Filter{{
		Patterns: []string{"*.mp3", ".flac", "*", "a/b"},
	}})
	require.Equal(t, "mp3,flac", got)
	require.Equal(t, "", extensionList(nil))
}

func TestListenerArgs(t *testing.T) {
	status, paths, err := listenerArgs([]any{"canceled", nil})
	require.NoError(t, err)
	require.Equal(t, "canceled", status)
	require.Equal(t, "", paths)

	_, _, err = listenerArgs([]any{"only"})
	require.ErrorIs(t, err, errDialog)
	_, _, err = listenerArgs([]any{1, "x"})
	require.ErrorIs(t, err, errDialog)
}

func TestResultOf(t *testing.T) {
	tests := []struct {
		name   string
		status string
		joined string
		want   []string
		err    error
	}{
		{name: "one", joined: "content://a", want: []string{"content://a"}},
		{name: "many", joined: "content://a\ncontent://b", want: []string{"content://a", "content://b"}},
		{name: "blank", joined: "content://a\n\ncontent://b", want: []string{"content://a", "content://b"}},
		{name: "cancel", status: statusCanceled, err: filedialog.ErrCanceled},
		{name: "no activity", status: statusNoActivity, err: driver.ErrUnavailable},
		{name: "no picker", status: statusNoPicker, err: driver.ErrUnavailable},
		{name: "empty", err: errURI},
		{name: "other", status: "boom", err: errDialog},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resultOf(tt.status, tt.joined)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPickWait(t *testing.T) {
	t.Cleanup(endPick)
	ch, gen, err := beginPick()
	require.NoError(t, err)
	_, _, err = beginPick()
	require.ErrorIs(t, err, errBusy)

	deliverPick(gen, statusCanceled, "")
	got := <-ch
	require.Equal(t, statusCanceled, got.status)

	endPick()
	deliverPick(gen, "late", "x")
	var late pick
	select {
	case late = <-ch:
	default:
	}
	require.Zero(t, late)

	ch, gen, err = beginPick()
	require.NoError(t, err)
	deliverPick(gen-1, "", "stale")
	deliverPick(gen, "", "content://a")
	got = <-ch
	require.Equal(t, "content://a", got.paths)
}
