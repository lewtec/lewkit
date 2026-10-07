// Package opengl loads OpenGL and OpenGL ES.
//
// Screen draws one GUI frame into a surface the caller owns. The frame
// matches the Vulkan swapchain draw: optional RGBA8 underlay, rounded-rect
// instances of 16 float32 values, then RGBA8 glyph ink. The shaders live in
// this package. The host window stays with the caller.
//
// One driver covers both APIs. Windows and Linux open desktop GL. Android
// opens OpenGL ES. Linux falls through to ES when desktop GL cannot create a
// context. The draw uses the shared 3.3 / ES 3.0 subset. Apple is Metal, so
// darwin and ios do not open a context here.
//
// Device runs one compute kernel. It does not attach a surface and it does
// not share the screen's context. Compute needs desktop GL 4.3 or OpenGL ES
// 3.1. ndarray talks to the device through a driver.
package opengl
