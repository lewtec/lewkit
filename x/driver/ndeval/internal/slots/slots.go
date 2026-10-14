// Package slots grows, fills, and reads the GPU buffers one kernel evaluation shares.
package slots

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver/ndeval/internal/pack"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Buffer is a device allocation the evaluators grow, fill, and read.
type Buffer interface {
	comparable
	Len() int
	Close() error
	Store(data []byte) error
	Read(dst []byte) error
}

// Set is the output, the inputs, and the binding list for one kernel.
type Set[B Buffer] struct {
	Output  B
	Inputs  []B
	Bound   []B
	Staging []byte
	Push    []byte
}

// CheckOutput reports a destination shorter than the kernel's host result.
// A zero-length kernel still fails here when the destination is short.
func CheckOutput(kernel *ndarray.Kernel, output []byte) error {
	hostBytes := kernel.OutputBytes()
	if len(output) < hostBytes {
		return fmt.Errorf("%w: output %d < %d", ndarray.ErrSize, len(output), hostBytes)
	}
	return nil
}

// Fit allocates the output and each input. A buffer already long enough stays.
// grew and total count only the allocations that replaced a slot.
func (set *Set[B]) Fit(kernel *ndarray.Kernel, alloc func(int) (B, error)) (grew, total int, err error) {
	if err := set.grow(&set.Output, max(kernel.Size(), 1)*4, alloc, &grew, &total); err != nil {
		return 0, 0, err
	}
	for i := 0; i < kernel.InputCount(); i++ {
		n := max(pack.Bytes(kernel.InputDType(i), len(kernel.Input(i))), 1)
		if err := set.grow(&set.Inputs[i], n, alloc, &grew, &total); err != nil {
			return 0, 0, err
		}
	}
	return grew, total, nil
}

// Load stores each input and sets Bound to the output followed by those inputs.
func (set *Set[B]) Load(kernel *ndarray.Kernel) error {
	for i := 0; i < kernel.InputCount(); i++ {
		if err := set.Inputs[i].Store(set.inputBytes(kernel, i)); err != nil {
			return err
		}
	}
	need := kernel.Bindings()
	if cap(set.Bound) < need {
		set.Bound = make([]B, need)
	} else {
		set.Bound = set.Bound[:need]
	}
	set.Bound[0] = set.Output
	copy(set.Bound[1:], set.Inputs)
	return nil
}

// FillPush writes the kernel push constants into Push and returns that slice.
func (set *Set[B]) FillPush(kernel *ndarray.Kernel) []byte {
	if cap(set.Push) < ndarray.PushBytes {
		set.Push = make([]byte, ndarray.PushBytes)
	} else {
		set.Push = set.Push[:ndarray.PushBytes]
	}
	kernel.FillPush(set.Push)
	return set.Push
}

// ReadHost copies the device result into output.
// A U8 kernel narrows little-endian uint32 words. Any other dtype copies the host bytes.
func (set *Set[B]) ReadHost(kernel *ndarray.Kernel, output []byte) error {
	hostBytes := kernel.OutputBytes()
	if kernel.DType() != ndarray.U8 {
		return set.Output.Read(output[:hostBytes])
	}
	size := kernel.Size()
	gpuBytes := size * 4
	if cap(set.Staging) < gpuBytes {
		set.Staging = make([]byte, gpuBytes)
	} else {
		set.Staging = set.Staging[:gpuBytes]
	}
	if err := set.Output.Read(set.Staging); err != nil {
		return err
	}
	for i := 0; i < size && i < len(output); i++ {
		output[i] = uint8(binary.LittleEndian.Uint32(set.Staging[i*4:]))
	}
	return nil
}

// Close closes the output and each input. A nil buffer is skipped.
func (set *Set[B]) Close() error {
	if set == nil {
		return nil
	}
	var zero B
	var err error
	if set.Output != zero {
		err = errors.Join(err, set.Output.Close())
		set.Output = zero
	}
	for i, buffer := range set.Inputs {
		if buffer == zero {
			continue
		}
		err = errors.Join(err, buffer.Close())
		set.Inputs[i] = zero
	}
	return err
}

func (set *Set[B]) grow(slot *B, bytes int, alloc func(int) (B, error), grew, total *int) error {
	var zero B
	cur := *slot
	if cur != zero && cur.Len() >= bytes {
		return nil
	}
	buffer, err := alloc(bytes)
	if err != nil {
		return err
	}
	*grew++
	*total += bytes
	if cur != zero {
		if err := cur.Close(); err != nil {
			return errors.Join(err, buffer.Close())
		}
	}
	*slot = buffer
	return nil
}

func (set *Set[B]) inputBytes(kernel *ndarray.Kernel, index int) []byte {
	data, staging := pack.Widen(kernel.InputDType(index), kernel.Input(index), set.Staging)
	set.Staging = staging
	return data
}
