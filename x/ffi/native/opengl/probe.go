//go:build !darwin && !ios

package opengl

import "encoding/binary"

// probeCompute compiles a one-thread kernel and checks the stored word.
// A context that claims compute but cannot run this shader is unavailable.
func probeCompute(device *Device) error {
	src := "#version 430\nlayout(local_size_x = 1) in;\nlayout(std430, binding = 0) buffer Out { uint o[]; };\nvoid main() { o[gl_GlobalInvocationID.x] = 7u; }\n"
	if device.GLES() {
		src = "#version 310 es\nprecision highp float;\nprecision highp int;\nlayout(local_size_x = 1) in;\nlayout(std430, binding = 0) buffer Out { uint o[]; };\nvoid main() { o[gl_GlobalInvocationID.x] = 7u; }\n"
	}
	prog, err := device.Compile(src)
	if err != nil {
		return err
	}
	defer prog.Close()
	buf, err := device.Buffer(4)
	if err != nil {
		return err
	}
	defer buf.Close()
	if err := device.Run(prog, 1, []*Buffer{buf}, nil); err != nil {
		return err
	}
	raw := make([]byte, 4)
	if err := buf.Read(raw); err != nil {
		return err
	}
	if binary.LittleEndian.Uint32(raw) != 7 {
		return ErrUnavailable
	}
	return nil
}
