package slots

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/ndarray"
)

type mem struct {
	n        int
	body     []byte
	readErr  error
	closeErr error
	closed   bool
}

func (m *mem) Len() int { return m.n }

func (m *mem) Store(data []byte) error {
	m.body = append([]byte(nil), data...)
	return nil
}

func (m *mem) Read(dst []byte) error {
	if m.readErr != nil {
		return m.readErr
	}
	copy(dst, m.body)
	return nil
}

func (m *mem) Close() error {
	m.closed = true
	return m.closeErr
}

func kernel(t *testing.T) *ndarray.Kernel {
	t.Helper()
	left, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	right, err := ndarray.New([]float32{5, 6, 7, 8}, ndarray.Shape{4})
	require.NoError(t, err)
	sum := left.Add(right)
	require.NoError(t, sum.Resize(sum.Shape()))
	require.NotNil(t, sum.Kernel())
	return sum.Kernel()
}

func u8Kernel(t *testing.T) *ndarray.Kernel {
	t.Helper()
	src, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{4})
	require.NoError(t, err)
	out := src.Cast[uint8]()
	require.NoError(t, out.Resize(out.Shape()))
	require.Equal(t, ndarray.U8, out.Kernel().DType())
	return out.Kernel()
}

func TestFitReusesALongBuffer(t *testing.T) {
	k := kernel(t)
	kept := &mem{n: 64}
	var allocs int
	set := Set[*mem]{Output: kept, Inputs: make([]*mem, k.InputCount())}
	grew, total, err := set.Fit(k, func(n int) (*mem, error) {
		allocs++
		return &mem{n: n}, nil
	})
	require.NoError(t, err)
	require.Same(t, kept, set.Output)
	require.False(t, kept.closed)
	require.Equal(t, 2, allocs)
	require.Equal(t, 2, grew)
	require.Equal(t, 32, total)
	require.Len(t, set.Inputs, 2)
	require.Equal(t, 16, set.Inputs[0].Len())
}

func TestGrowJoinsACloseError(t *testing.T) {
	closeErr := errors.New("close")
	old := &mem{n: 1, closeErr: closeErr}
	set := Set[*mem]{Output: old}
	var grew, total int
	err := set.grow(&set.Output, 8, func(n int) (*mem, error) {
		return &mem{n: n}, nil
	}, &grew, &total)
	require.ErrorIs(t, err, closeErr)
	require.True(t, old.closed)
	require.Same(t, old, set.Output)
}

func TestLoadAndReadHost(t *testing.T) {
	k := kernel(t)
	set := Set[*mem]{Inputs: make([]*mem, k.InputCount())}
	_, _, err := set.Fit(k, func(n int) (*mem, error) {
		return &mem{n: n}, nil
	})
	require.NoError(t, err)
	require.NoError(t, set.Load(k))
	require.Len(t, set.Bound, k.Bindings())
	require.Same(t, set.Output, set.Bound[0])
	require.NotEmpty(t, set.Inputs[0].body)

	host := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	set.Output.body = host
	out := make([]byte, k.OutputBytes())
	require.NoError(t, set.ReadHost(k, out))
	require.Equal(t, host, out)

	push := set.FillPush(k)
	require.Len(t, push, ndarray.PushBytes)
	require.Equal(t, uint32(k.Size()), binary.LittleEndian.Uint32(push[:4]))

	u8 := u8Kernel(t)
	words := make([]byte, u8.Size()*4)
	binary.LittleEndian.PutUint32(words[0:], 9)
	binary.LittleEndian.PutUint32(words[4:], 8)
	narrow := Set[*mem]{Output: &mem{body: words}}
	u8Out := make([]byte, u8.OutputBytes())
	require.NoError(t, narrow.ReadHost(u8, u8Out))
	require.Equal(t, []byte{9, 8, 0, 0}, u8Out)
}

func TestCheckOutput(t *testing.T) {
	k := kernel(t)
	err := CheckOutput(k, []byte{1})
	require.ErrorIs(t, err, ndarray.ErrSize)
	require.NoError(t, CheckOutput(k, make([]byte, k.OutputBytes())))
}

func TestCloseSkipsNil(t *testing.T) {
	live := &mem{}
	set := Set[*mem]{Output: live, Inputs: []*mem{nil, live}}
	require.NoError(t, set.Close())
	require.True(t, live.closed)
	require.Nil(t, set.Output)
}
