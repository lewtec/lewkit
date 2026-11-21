// Package vulkan loads libvulkan through [github.com/lewtec/lewkit/x/ffi/native]
// and runs compute shaders.
//
//	d, err := vulkan.Open(ctx)
//	buf, err := d.Buffer(256)
//	sh, err := d.Shader(ctx, spirv, 1)
//	err = d.Run(sh, 4, 1, 1, buf)
//
//	cmd, err := d.Begin()
//	cmd.Bind(sh, weights, acts)
//	cmd.Push(shape)
//	cmd.Dispatch(gx, gy, 1)
//	cmd.Barrier()
//	cmd.Submit() // signals a fence
//	cmd.Wait()   // vkWaitForFences; FFI syscall, GC-safe while GPU runs
//
// [Open] dlopens the loader and resolves commands with
// vkGetInstanceProcAddr. Most vk* names are not loader exports.
//
// This is a compute subset, not a generated dump of vulkan.h.
// Khronos generates the C headers from vk.xml; lewkit has no
// C-header bindgen.
package vulkan
