package opengl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/driver/ndeval/spell"
	"github.com/lewtec/lewkit/x/ndarray"
)

// The body uses the GLSL tokens. The header is desktop GL 4.3 or ES 3.1.
var glslBody = spell.Dialect{
	NonFinite: "uintBitsToFloat(%du)",
	MinInt32:  "int(0x80000000u)",
}

// glslSource renders the kernel schedule as a desktop GL 4.3 or ES 3.1 compute shader.
// gles selects the ES header.
func glslSource(k *ndarray.Kernel, gles bool) (string, error) {
	if k == nil {
		return "", ndarray.ErrOp
	}
	code, err := k.Code()
	if err != nil {
		return "", err
	}
	return glslOf(code, gles), nil
}

func glslOf(c ndarray.Code, gles bool) string {
	var b strings.Builder
	if gles {
		b.WriteString("#version 310 es\nprecision highp float;\nprecision highp int;\n")
	} else {
		b.WriteString("#version 430\n")
	}
	b.WriteString("layout(local_size_x = ")
	b.WriteString(strconv.Itoa(c.Threads))
	b.WriteString(") in;\n")
	b.WriteString("uniform uint n;\nuniform uint d0;\nuniform uint d1;\nuniform uint d2;\nuniform uint d3;\n")
	fmt.Fprintf(&b, "layout(std430, binding = 0) buffer Out { %s o[]; };\n", spell.Scalar(c.Out))
	for i, dt := range c.In {
		fmt.Fprintf(&b, "layout(std430, binding = %d) buffer In%d { %s x%d[]; };\n", i+1, i+1, spell.Scalar(dt), i+1)
	}
	b.WriteString("void main() {\n")
	// X alone is at most GL_MAX_COMPUTE_WORK_GROUP_COUNT, which is 65535
	// on Adreno. A phone frame needs more groups, so Run may use Y and Z.
	// gi stays the linear element index either way.
	b.WriteString("    uint span = gl_NumWorkGroups.x * gl_WorkGroupSize.x;\n")
	b.WriteString("    uint gi = gl_GlobalInvocationID.x + gl_GlobalInvocationID.y * span + gl_GlobalInvocationID.z * span * gl_NumWorkGroups.y;\n")
	spell.WriteBody(&b, c, glslBody)
	b.WriteString("}\n")
	return b.String()
}
