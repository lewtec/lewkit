package native

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"unsafe"

	"github.com/stretchr/testify/require"
)

var (
	errMissing = errors.New("missing")
	errBoom    = errors.New("boom")
)

func swapLoader(t *testing.T, fn func(string, int) (uintptr, error)) {
	t.Helper()
	prev := loadLibrary
	loadLibrary = fn
	t.Cleanup(func() { loadLibrary = prev })
}

func TestOpenSharesOneLoad(t *testing.T) {
	var n atomic.Int32
	swapLoader(t, func(string, int) (uintptr, error) {
		n.Add(1)
		return 11, nil
	})
	path := "lewkit.test/once"
	first, err := Open(path, Lazy)
	require.NoError(t, err)
	second, err := Open(path, Lazy)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, int32(1), n.Load())
}

func TestOpenFailureIsNotCached(t *testing.T) {
	var n atomic.Int32
	swapLoader(t, func(string, int) (uintptr, error) {
		if n.Add(1) == 1 {
			return 0, errMissing
		}
		return 7, nil
	})
	path := "lewkit.test/retry"
	_, err := Open(path, Lazy)
	require.EqualError(t, err, "missing")
	handle, err := Open(path, Lazy)
	require.NoError(t, err)
	require.Equal(t, uintptr(7), handle)
	require.Equal(t, int32(2), n.Load())
}

func TestOpenConcurrentCallersShareOneLoad(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var n atomic.Int32
	swapLoader(t, func(string, int) (uintptr, error) {
		if n.Add(1) == 1 {
			close(started)
			<-release
		}
		return 3, nil
	})
	path := "lewkit.test/flight"
	type result struct {
		handle uintptr
		err    error
	}
	out := make(chan result, 8)
	for range 8 {
		go func() {
			handle, err := Open(path, Lazy)
			out <- result{handle, err}
		}()
	}
	<-started
	close(release)
	for range 8 {
		got := <-out
		require.NoError(t, got.err)
		require.Equal(t, uintptr(3), got.handle)
	}
	require.Equal(t, int32(1), n.Load())
}

func TestOpenConcurrentFailureIsShared(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		var n atomic.Int32
		swapLoader(t, func(string, int) (uintptr, error) {
			n.Add(1)
			<-release
			return 0, errMissing
		})
		path := "lewkit.test/flight-miss"
		errc := make(chan error)
		for range 8 {
			go func() {
				_, err := Open(path, Lazy)
				errc <- err
			}()
		}
		synctest.Wait()
		require.Equal(t, int32(1), n.Load())
		close(release)
		for range 8 {
			require.EqualError(t, <-errc, "missing")
		}
		require.Equal(t, int32(1), n.Load())
	})
}

func TestOpenPromotesFlags(t *testing.T) {
	var got []int
	swapLoader(t, func(_ string, flags int) (uintptr, error) {
		got = append(got, flags)
		return uintptr(len(got)), nil
	})
	path := "lewkit.test/flags"
	first, err := Open(path, Lazy)
	require.NoError(t, err)
	again, err := Open(path, Lazy)
	require.NoError(t, err)
	require.Equal(t, first, again)
	promoted, err := Open(path, Global|Lazy)
	require.NoError(t, err)
	require.NotEqual(t, first, promoted)
	covered, err := Open(path, Lazy)
	require.NoError(t, err)
	require.Equal(t, promoted, covered)
	require.Equal(t, []int{Lazy, Global | Lazy}, got)
}

func TestOpenFirst(t *testing.T) {
	swapLoader(t, func(path string, _ int) (uintptr, error) {
		if path == "good" {
			return 9, nil
		}
		return 0, fmt.Errorf("%s: %w", path, errMissing)
	})
	_, err := OpenFirst(Lazy)
	require.EqualError(t, err, "no library path")
	_, err = OpenFirst(Lazy, "a", "b")
	require.EqualError(t, err, "b: missing")
	handle, err := OpenFirst(Lazy, "a", "good", "c")
	require.NoError(t, err)
	require.Equal(t, uintptr(9), handle)
}

func TestOnceKeepsTheFirstError(t *testing.T) {
	var n atomic.Int32
	load := Once(func() error {
		n.Add(1)
		return errBoom
	})
	out := make(chan error)
	for range 8 {
		go func() { out <- load() }()
	}
	for range 8 {
		require.EqualError(t, <-out, "boom")
	}
	require.EqualError(t, load(), "boom")
	require.Equal(t, int32(1), n.Load())
}

func TestCStringAndGoString(t *testing.T) {
	raw := CString("pulse")
	require.Equal(t, []byte{'p', 'u', 'l', 's', 'e', 0}, raw)
	require.Equal(t, "pulse", GoString(uintptr(unsafe.Pointer(&raw[0]))))
	require.Equal(t, "", GoString(0))
	require.Equal(t, []byte{0}, CString(""))
}
