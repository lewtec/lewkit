//go:build !windows || (!amd64 && !arm64)

package d3d12

// Screen is the Direct3D 12 drawable for one HWND. This build has no Direct3D 12.
type Screen struct{}

// Device is the Direct3D 12 compute device. This build has no Direct3D 12.
type Device struct{}

// Pipeline is one compiled compute kernel. This build has no Direct3D 12.
type Pipeline struct{}

// Buffer is a storage buffer. This build has no Direct3D 12.
type Buffer struct{}

// Available reports that Direct3D 12 is not on this OS.
func Available() error { return ErrUnavailable }

// OpenNative reports that Direct3D 12 is not on this OS.
// kind 2 is a Win32 HWND. This build does not open it.
func OpenNative(int, uintptr, int, int) (*Screen, error) {
	return nil, ErrUnavailable
}

// Draw reports that the screen is closed.
func (*Screen) Draw([]byte, []byte, []byte, int, int) error { return ErrUnavailable }

// Adopt reports that the screen is closed.
func (*Screen) Adopt(uintptr, int, int) error { return ErrUnavailable }

// Close is a no-op.
func (*Screen) Close() error { return nil }

// OpenDevice reports that Direct3D 12 compute is not on this OS.
func OpenDevice() (*Device, error) { return nil, ErrUnavailable }

// Name is empty on this build.
func (*Device) Name() string { return "" }

// Close is a no-op.
func (*Device) Close() error { return nil }

// Buffer reports that Direct3D 12 compute is not on this OS.
func (*Device) Buffer(int) (*Buffer, error) { return nil, ErrUnavailable }

// Compile reports that Direct3D 12 compute is not on this OS.
func (*Device) Compile(string, int) (*Pipeline, error) { return nil, ErrUnavailable }

// Len is zero.
func (*Buffer) Len() int { return 0 }

// Store reports that Direct3D 12 compute is not on this OS.
func (*Buffer) Store([]byte) error { return ErrUnavailable }

// Read reports that Direct3D 12 compute is not on this OS.
func (*Buffer) Read([]byte) error { return ErrUnavailable }

// Close is a no-op.
func (*Buffer) Close() error { return nil }

// Dispatch reports that Direct3D 12 compute is not on this OS.
func (*Pipeline) Dispatch([]*Buffer, []byte, int) error { return ErrUnavailable }

// Close is a no-op.
func (*Pipeline) Close() error { return nil }
