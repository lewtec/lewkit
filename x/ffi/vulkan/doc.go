// Package vulkan loads libvulkan through [github.com/lewtec/lewkit/x/ffi]
// and runs compute shaders.
//
//	d, err := vulkan.Open(ctx)
//	buf, err := d.Buffer(256)
//	sh, err := d.Shader(ctx, src, 1) // SPIR-V or Vulkan GLSL
//	err = d.Run(sh, 4, 1, 1, buf)
//
// [Open] dlopens the loader and resolves commands with
// vkGetInstanceProcAddr. Most vk* names are not loader exports.
//
// This is a compute subset, not a generated dump of vulkan.h.
// Khronos generates the C headers from vk.xml; lewkit has no
// C-header bindgen.
package vulkan
