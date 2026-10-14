package table

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/ndarray"
)

type program struct {
	kernel *ndarray.Kernel
	closed int
	fail   error
}

func (p *program) Close() error {
	p.closed++
	return p.fail
}

func TestOpenKeepsTheFirstProgram(t *testing.T) {
	kernel := new(ndarray.Kernel)
	var programs Table[*program]
	same := func(p *program) bool { return p.kernel == kernel }
	first, fresh, err := programs.Open(kernel, same, func() (*program, error) {
		return &program{kernel: kernel}, nil
	}, nil)
	require.NoError(t, err)
	require.True(t, fresh)

	second, fresh, err := programs.Open(kernel, same, func() (*program, error) {
		require.FailNow(t, "create")
		return nil, nil
	}, nil)
	require.NoError(t, err)
	require.False(t, fresh)
	require.Same(t, first, second)
	require.Equal(t, 0, second.closed)
}

func TestInstallClosesTheLoser(t *testing.T) {
	kernel := new(ndarray.Kernel)
	var programs Table[*program]
	first := &program{kernel: kernel}
	programs.items = map[*ndarray.Kernel]*program{kernel: first}
	loser := &program{kernel: kernel}
	same := func(p *program) bool { return p.kernel == kernel }
	kept, err := programs.install(kernel, loser, same)
	require.NoError(t, err)
	require.Same(t, first, kept)
	require.Equal(t, 1, loser.closed)
	require.Equal(t, 0, first.closed)
}

func TestOpenReplacesAStaleProgram(t *testing.T) {
	kernel := new(ndarray.Kernel)
	var programs Table[*program]
	other := new(ndarray.Kernel)
	stale := &program{kernel: other}
	programs.items = map[*ndarray.Kernel]*program{kernel: stale}
	same := func(p *program) bool { return p.kernel == kernel }
	fresh, created, err := programs.Open(kernel, same, func() (*program, error) {
		return &program{kernel: kernel}, nil
	}, func(p *program) {
		p.kernel = kernel
	})
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, 1, stale.closed)
	require.Same(t, fresh, programs.items[kernel])
}

func TestInstallReturnsTheCloseError(t *testing.T) {
	kernel := new(ndarray.Kernel)
	var programs Table[*program]
	programs.items = map[*ndarray.Kernel]*program{}
	closeErr := errors.New("close")
	old := &program{kernel: new(ndarray.Kernel), fail: closeErr}
	programs.items[kernel] = old
	same := func(p *program) bool { return p.kernel == kernel }
	_, err := programs.install(kernel, &program{kernel: kernel}, same)
	require.ErrorIs(t, err, closeErr)
	require.Equal(t, kernel, programs.items[kernel].kernel)
}

func TestForgetAndTake(t *testing.T) {
	kernel := new(ndarray.Kernel)
	var programs Table[*program]
	prog := &program{kernel: kernel}
	programs.items = map[*ndarray.Kernel]*program{kernel: prog}
	programs.Forget(kernel, &program{})()
	require.Same(t, prog, programs.items[kernel])
	programs.Forget(kernel, prog)()
	_, ok := programs.items[kernel]
	require.False(t, ok)

	programs.items = map[*ndarray.Kernel]*program{kernel: prog}
	var cleared bool
	list := programs.Take(func() { cleared = true })
	require.True(t, cleared)
	require.Nil(t, programs.items)
	require.Equal(t, []*program{prog}, list)
	programs.Forget(kernel, prog)()
}
