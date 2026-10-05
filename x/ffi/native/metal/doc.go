// Package metal loads Metal.
//
// Screen draws one GUI frame into a caller-owned view. The frame matches
// the Vulkan swapchain draw: optional RGBA8 underlay, rounded-rect instances
// of 16 float32 values, then RGBA8 glyph ink. The shaders live in this package.
// The host window stays with the caller.
//
// Device runs one compute kernel. It does not attach a view and it does not
// share the screen's queue. ndarray talks to it through a driver.
package metal
