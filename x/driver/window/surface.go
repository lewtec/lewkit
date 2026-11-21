package window

// Surface is the native window a Vulkan swapchain presents into.
// The window keeps events and resize. Vulkan only borrows this handle.
type Surface struct {
	Kind int
	A, B uintptr
}

const (
	// SurfaceX11 is an Xlib Display* in A and a Window XID in B.
	SurfaceX11 = 1
	// SurfaceWin32 is an HWND in A and an HINSTANCE in B.
	SurfaceWin32 = 2
	// SurfaceView is an NSView* in A. Vulkan attaches a CAMetalLayer.
	SurfaceView = 3
)

// Surfacer is a host window that can lend its native handle.
type Surfacer interface {
	Surface() Surface
}
