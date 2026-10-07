// Package d3d12 loads Direct3D 12.
//
// Screen draws one GUI frame into a caller-owned HWND. The frame matches
// the Vulkan swapchain draw: optional RGBA8 underlay, rounded-rect instances
// of 16 float32 values, then RGBA8 glyph ink. The shaders live in this package.
// The host window stays with the caller.
//
// Device runs one compute kernel. It does not attach a window and it does not
// share the screen's queue. ndarray talks to it through a driver.
package d3d12
