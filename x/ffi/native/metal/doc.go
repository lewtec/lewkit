// Package metal loads Metal and draws one GUI frame into a caller-owned view.
//
// The frame matches the Vulkan swapchain draw: optional RGBA8 underlay,
// rounded-rect instances of 16 float32 values, then RGBA8 glyph ink.
// The shaders live in this package. The host window stays with the caller.
package metal
