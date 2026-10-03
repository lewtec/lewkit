package window

// Surface is the native window a present screen draws into.
// The window keeps events and resize. The screen only borrows this handle.
type Surface struct {
	Kind int
	A, B uintptr
}

const (
	// SurfaceX11 is an Xlib Display* in A and a Window XID in B.
	SurfaceX11 = 1
	// SurfaceWin32 is an HWND in A and an HINSTANCE in B.
	SurfaceWin32 = 2
	// SurfaceView is an NSView* in A. Vulkan or Metal attaches a CAMetalLayer.
	SurfaceView = 3
	// SurfaceAndroid is an ANativeWindow* in A.
	SurfaceAndroid = 4
	// SurfaceUIView is a UIView* in A. Metal attaches a CAMetalLayer.
	SurfaceUIView = 5
)

// Surfacer is a host window that can lend its native handle.
type Surfacer interface {
	Surface() Surface
}
