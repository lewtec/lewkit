// Package vulkan is the ndarray evaluator backed by a Vulkan device.
// It renders the kernel schedule as GLSL. A registered compute hash is
// stored SPIR-V. A miss uses the glslc driver: native glslc when it is
// on PATH, otherwise the lewkit reactor. Draw shaders stay in the
// x/driver/vulkan registry. Bind attaches a caller-owned device. Paint
// presents a mounted tensor on that device.
// The driver id ndeval_vulkan:<type> opens that device class for the
// ndeval expression. Its weight is 0, so ndarray.Open still uses ndeval_vulkan.
// x/ndarray does not import this package.
package vulkan
