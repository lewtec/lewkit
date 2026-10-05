//go:build !darwin

package metal

// Device is the Metal compute device. This build has no Metal.
type Device struct{}

// Pipeline is one compiled compute kernel. This build has no Metal.
type Pipeline struct{}

// Buffer is a shared storage buffer. This build has no Metal.
type Buffer struct{}

// OpenDevice reports that Metal compute is not on this OS.
func OpenDevice() (*Device, error) { return nil, ErrUnavailable }

// Name is empty on this build.
func (*Device) Name() string { return "" }

// Close is a no-op.
func (*Device) Close() error { return nil }

// Buffer reports that Metal compute is not on this OS.
func (*Device) Buffer(int) (*Buffer, error) { return nil, ErrUnavailable }

// Compile reports that Metal compute is not on this OS.
func (*Device) Compile(string, int) (*Pipeline, error) { return nil, ErrUnavailable }

// Len is zero.
func (*Buffer) Len() int { return 0 }

// Store reports that Metal compute is not on this OS.
func (*Buffer) Store([]byte) error { return ErrUnavailable }

// Read reports that Metal compute is not on this OS.
func (*Buffer) Read([]byte) error { return ErrUnavailable }

// Close is a no-op.
func (*Buffer) Close() error { return nil }

// Dispatch reports that Metal compute is not on this OS.
func (*Pipeline) Dispatch([]*Buffer, []byte, int) error { return ErrUnavailable }

// Close is a no-op.
func (*Pipeline) Close() error { return nil }
