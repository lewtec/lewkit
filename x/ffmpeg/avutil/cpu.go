package avutil

import "golang.org/x/sys/cpu"

// CPU is the instruction sets this process may use.
type CPU struct {
	SSE2  bool
	SSSE3 bool
	SSE41 bool
	AVX   bool
	AVX2  bool
}

// scalar forces the C kernels. ffmpeg -cpuflags 0 sets it.
var scalar bool

// SetScalar disables assembly kernels when on is true.
func SetScalar(on bool) { scalar = on }

// Scalar reports whether assembly kernels are off.
func Scalar() bool { return scalar }

// HostCPU reads the processor flags once.
func HostCPU() CPU {
	if scalar {
		return CPU{}
	}
	return CPU{
		SSE2:  cpu.X86.HasSSE2,
		SSSE3: cpu.X86.HasSSSE3,
		SSE41: cpu.X86.HasSSE41,
		AVX:   cpu.X86.HasAVX,
		AVX2:  cpu.X86.HasAVX2,
	}
}

// String lists the flags ffmpeg's -cpuflags style would print for this host.
func (c CPU) String() string {
	var parts []string
	if c.SSE2 {
		parts = append(parts, "sse2")
	}
	if c.SSSE3 {
		parts = append(parts, "ssse3")
	}
	if c.SSE41 {
		parts = append(parts, "sse4.1")
	}
	if c.AVX {
		parts = append(parts, "avx")
	}
	if c.AVX2 {
		parts = append(parts, "avx2")
	}
	if len(parts) == 0 {
		return "none"
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += " " + part
	}
	return out
}
