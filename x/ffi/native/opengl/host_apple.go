//go:build darwin || ios

package opengl

// Apple presents and evaluates with Metal. These types exist so the package
// compiles. Every entry point returns ErrUnavailable.

// Screen is unused on Apple.
type Screen struct{}

// Draw reports that Apple does not use OpenGL.
func (*Screen) Draw([]byte, []byte, []byte, int, int) error { return ErrUnavailable }

// Read reports that Apple does not use OpenGL.
func (*Screen) Read() ([]byte, error) { return nil, ErrUnavailable }

// Adopt reports that Apple does not use OpenGL.
func (*Screen) Adopt(uintptr, int, int) error { return ErrUnavailable }

// Close reports that Apple does not use OpenGL.
func (*Screen) Close() error { return nil }

// Device is unused on Apple.
type Device struct{}

// Name is the driver label.
func (*Device) Name() string { return "opengl" }

// GLES reports that Apple does not use OpenGL ES here.
func (*Device) GLES() bool { return false }

// Buffer reports that Apple does not use OpenGL.
func (*Device) Buffer(int) (*Buffer, error) { return nil, ErrUnavailable }

// Compile reports that Apple does not use OpenGL.
func (*Device) Compile(string) (*Program, error) { return nil, ErrUnavailable }

// Run reports that Apple does not use OpenGL.
func (*Device) Run(*Program, uint32, []*Buffer, []byte) error { return ErrUnavailable }

// Close reports that Apple does not use OpenGL.
func (*Device) Close() error { return nil }

// Buffer is unused on Apple.
type Buffer struct{}

// Len is zero.
func (*Buffer) Len() int { return 0 }

// Store reports that Apple does not use OpenGL.
func (*Buffer) Store([]byte) error { return ErrUnavailable }

// Read reports that Apple does not use OpenGL.
func (*Buffer) Read([]byte) error { return ErrUnavailable }

// Close reports that Apple does not use OpenGL.
func (*Buffer) Close() error { return nil }

// Program is unused on Apple.
type Program struct{}

// Close reports that Apple does not use OpenGL.
func (*Program) Close() error { return nil }

// OpenNative reports that Apple does not use OpenGL.
func OpenNative(int, uintptr, uintptr, int, int) (*Screen, error) {
	return nil, ErrUnavailable
}

// OpenOffscreen reports that Apple does not use OpenGL.
func OpenOffscreen(int, int) (*Screen, error) { return nil, ErrUnavailable }

// OpenDevice reports that Apple does not use OpenGL.
func OpenDevice() (*Device, error) { return nil, ErrUnavailable }

// Available reports that Apple does not use OpenGL.
func Available() error { return ErrUnavailable }

// ComputeAvailable reports that Apple does not use OpenGL.
func ComputeAvailable() error { return ErrUnavailable }
