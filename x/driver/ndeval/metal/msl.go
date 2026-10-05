package metal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/ndarray"
)

// mslSource lowers the ndarray GLSL dialect to one Metal compute kernel.
// The dialect is what Kernel.GLSL emits: one 1D workgroup, a 20-byte push
// constant, and storage buffers. The ndarray package stays free of Metal.
func mslSource(src string) (string, int, error) {
	const version = "#version 450\n"
	const push = "layout(push_constant) uniform Push { uint n; uint d0; uint d1; uint d2; uint d3; };\n"
	const main = "void main() {\n"
	if !strings.HasPrefix(src, version) {
		return "", 0, fmt.Errorf("%w: glsl version", ndarray.ErrOp)
	}
	rest := src[len(version):]
	line, rest, ok := strings.Cut(rest, "\n")
	if !ok {
		return "", 0, fmt.Errorf("%w: glsl local size", ndarray.ErrOp)
	}
	threads, err := parseLocalSize(line)
	if err != nil {
		return "", 0, err
	}
	if !strings.HasPrefix(rest, push) {
		return "", 0, fmt.Errorf("%w: glsl push", ndarray.ErrOp)
	}
	rest = rest[len(push):]
	var bufs []mslBuf
	for !strings.HasPrefix(rest, main) {
		line, next, ok := strings.Cut(rest, "\n")
		if !ok {
			return "", 0, fmt.Errorf("%w: glsl buffer", ndarray.ErrOp)
		}
		buf, err := parseBuffer(line)
		if err != nil {
			return "", 0, err
		}
		if buf.bind != len(bufs) {
			return "", 0, fmt.Errorf("%w: glsl binding %d", ndarray.ErrOp, buf.bind)
		}
		bufs = append(bufs, buf)
		rest = next
	}
	if len(bufs) < 1 {
		return "", 0, fmt.Errorf("%w: glsl output", ndarray.ErrOp)
	}
	body := rest[len(main):]
	if !strings.HasSuffix(body, "}\n") {
		return "", 0, fmt.Errorf("%w: glsl main", ndarray.ErrOp)
	}
	body = body[:len(body)-len("}\n")]
	const gid = "    uint gi = gl_GlobalInvocationID.x;\n"
	if strings.Count(body, gid) != 1 {
		return "", 0, fmt.Errorf("%w: glsl invocation", ndarray.ErrOp)
	}
	body = strings.Replace(body, gid, "    uint gi = gid;\n", 1)
	if strings.Contains(body, "gl_") || strings.Contains(body, "layout(") {
		return "", 0, fmt.Errorf("%w: glsl builtin", ndarray.ErrOp)
	}
	return emitMSL(bufs, body), threads, nil
}

type mslBuf struct {
	bind int
	typ  string
	name string
}

func parseLocalSize(line string) (int, error) {
	const prefix = "layout(local_size_x = "
	const suffix = ") in;"
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		return 0, fmt.Errorf("%w: glsl local size", ndarray.ErrOp)
	}
	n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix))
	if err != nil || n < 1 || n > 1024 {
		return 0, fmt.Errorf("%w: glsl local size", ndarray.ErrOp)
	}
	return n, nil
}

var bufferLine = regexp.MustCompile(`^layout\(set = 0, binding = ([0-9]+)\) buffer [A-Za-z0-9]+ \{ (float|int|uint) ([A-Za-z_][A-Za-z0-9]*)\[\]; \};$`)

func parseBuffer(line string) (mslBuf, error) {
	m := bufferLine.FindStringSubmatch(line)
	if m == nil {
		return mslBuf{}, fmt.Errorf("%w: glsl buffer %q", ndarray.ErrOp, line)
	}
	bind, err := strconv.Atoi(m[1])
	if err != nil {
		return mslBuf{}, fmt.Errorf("%w: glsl buffer", ndarray.ErrOp)
	}
	return mslBuf{bind: bind, typ: m[2], name: m[3]}, nil
}

func emitMSL(bufs []mslBuf, body string) string {
	var b strings.Builder
	b.WriteString("#include <metal_stdlib>\nusing namespace metal;\n\n")
	b.WriteString("struct Push {\n    uint n;\n    uint d0;\n    uint d1;\n    uint d2;\n    uint d3;\n};\n\n")
	b.WriteString("kernel void ndeval(\n")
	for _, buf := range bufs {
		fmt.Fprintf(&b, "    device %s* %s [[buffer(%d)]],\n", buf.typ, buf.name, buf.bind)
	}
	fmt.Fprintf(&b, "    constant Push& push [[buffer(%d)]],\n", len(bufs))
	b.WriteString("    uint gid [[thread_position_in_grid]])\n{\n")
	b.WriteString("    uint n = push.n;\n")
	b.WriteString("    uint d0 = push.d0;\n")
	b.WriteString("    uint d1 = push.d1;\n")
	b.WriteString("    uint d2 = push.d2;\n")
	b.WriteString("    uint d3 = push.d3;\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("}\n")
	return b.String()
}
