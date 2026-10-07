//go:build !darwin && !ios

package opengl

import (
	"fmt"
	"runtime"
	"strings"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

const (
	glColorBufferBit          = 0x00004000
	glBlend                   = 0x0BE2
	glOne                     = 1
	glOneMinusSrcAlpha        = 0x0303
	glDepthTest               = 0x0B71
	glCullFace                = 0x0B44
	glScissorTest             = 0x0C11
	glStencilTest             = 0x0B90
	glDither                  = 0x0BD0
	glMultisample             = 0x809D
	glTexture2D               = 0x0DE1
	glTexture0                = 0x84C0
	glTextureMinFilter        = 0x2801
	glTextureMagFilter        = 0x2800
	glTextureWrapS            = 0x2802
	glTextureWrapT            = 0x2803
	glNearest                 = 0x2600
	glClampToEdge             = 0x812F
	glUnpackAlignment         = 0x0CF5
	glPackAlignment           = 0x0D05
	glRGBA                    = 0x1908
	glRGBA8                   = 0x8058
	glUnsignedByte            = 0x1401
	glFloat                   = 0x1406
	glTriangleStrip           = 0x0005
	glArrayBuffer             = 0x8892
	glDynamicDraw             = 0x88E8
	glFragmentShader          = 0x8B30
	glVertexShader            = 0x8B31
	glComputeShader           = 0x91B9
	glCompileStatus           = 0x8B81
	glLinkStatus              = 0x8B82
	glInfoLogLength           = 0x8B84
	glFramebuffer             = 0x8D40
	glColorAttachment0        = 0x8CE0
	glFramebufferComplete     = 0x8CD5
	glMajorVersion            = 0x821B
	glMinorVersion            = 0x821C
	glVersion                 = 0x1F02
	glRenderer                = 0x1F01
	glShaderStorageBuffer     = 0x90D2
	glMapReadBit              = 0x0001
	glMapWriteBit             = 0x0002
	glMapInvalidateBufferBit  = 0x0008
	glShaderStorageBarrierBit = 0x00002000
	glBufferUpdateBarrierBit  = 0x00000200
)

type glAPI struct {
	gles             bool
	major, minor     int32
	renderer         string
	compute          bool
	getError         func() uint32
	getIntegerv      func(uint32, *int32)
	getString        func(uint32) uintptr
	viewport         func(int32, int32, int32, int32)
	clearColor       func(float32, float32, float32, float32)
	clear            func(uint32)
	enable           func(uint32)
	disable          func(uint32)
	blendFunc        func(uint32, uint32)
	genVertexArrays  func(int32, *uint32)
	bindVertexArray  func(uint32)
	delVertexArrays  func(int32, *uint32)
	genBuffers       func(int32, *uint32)
	bindBuffer       func(uint32, uint32)
	bufferData       func(uint32, int, uintptr, uint32)
	deleteBuffers    func(int32, *uint32)
	genTextures      func(int32, *uint32)
	bindTexture      func(uint32, uint32)
	texParameteri    func(uint32, uint32, int32)
	texImage2D       func(uint32, int32, int32, int32, int32, int32, uint32, uint32, uintptr)
	pixelStorei      func(uint32, int32)
	activeTexture    func(uint32)
	genFramebuffers  func(int32, *uint32)
	bindFramebuffer  func(uint32, uint32)
	frameTex2D       func(uint32, uint32, uint32, uint32, int32)
	checkFramebuffer func(uint32) uint32
	delFramebuffers  func(int32, *uint32)
	deleteTextures   func(int32, *uint32)
	createShader     func(uint32) uint32
	shaderSource     func(uint32, int32, uintptr, uintptr)
	compileShader    func(uint32)
	getShaderiv      func(uint32, uint32, *int32)
	shaderInfoLog    func(uint32, int32, uintptr, uintptr)
	createProgram    func() uint32
	attachShader     func(uint32, uint32)
	linkProgram      func(uint32)
	getProgramiv     func(uint32, uint32, *int32)
	programInfoLog   func(uint32, int32, uintptr, uintptr)
	useProgram       func(uint32)
	deleteShader     func(uint32)
	deleteProgram    func(uint32)
	uniformLocation  func(uint32, uintptr) int32
	uniform1i        func(int32, int32)
	uniform1ui       func(int32, uint32)
	uniform2f        func(int32, float32, float32)
	enableAttrib     func(uint32)
	disableAttrib    func(uint32)
	attribPointer    func(uint32, int32, uint32, uint8, int32, uintptr)
	attribDivisor    func(uint32, uint32)
	drawArrays       func(uint32, int32, int32)
	drawInstanced    func(uint32, int32, int32, int32)
	readPixels       func(int32, int32, int32, int32, uint32, uint32, uintptr)
	finish           func()
	dispatch         func(uint32, uint32, uint32)
	barrier          func(uint32)
	bindBufferBase   func(uint32, uint32, uint32)
	mapBuffer        func(uint32, int, int, uint32) uintptr
	unmapBuffer      func(uint32) uint8
	getIntegeri      func(uint32, uint32, *int32)
}

func loadAPI(ctx glContext) (*glAPI, error) {
	g := &glAPI{}
	need := []struct {
		name string
		fn   any
	}{
		{"glGetError", &g.getError},
		{"glGetIntegerv", &g.getIntegerv},
		{"glGetString", &g.getString},
		{"glViewport", &g.viewport},
		{"glClearColor", &g.clearColor},
		{"glClear", &g.clear},
		{"glEnable", &g.enable},
		{"glDisable", &g.disable},
		{"glBlendFunc", &g.blendFunc},
		{"glGenVertexArrays", &g.genVertexArrays},
		{"glBindVertexArray", &g.bindVertexArray},
		{"glDeleteVertexArrays", &g.delVertexArrays},
		{"glGenBuffers", &g.genBuffers},
		{"glBindBuffer", &g.bindBuffer},
		{"glBufferData", &g.bufferData},
		{"glDeleteBuffers", &g.deleteBuffers},
		{"glGenTextures", &g.genTextures},
		{"glBindTexture", &g.bindTexture},
		{"glTexParameteri", &g.texParameteri},
		{"glTexImage2D", &g.texImage2D},
		{"glPixelStorei", &g.pixelStorei},
		{"glActiveTexture", &g.activeTexture},
		{"glGenFramebuffers", &g.genFramebuffers},
		{"glBindFramebuffer", &g.bindFramebuffer},
		{"glFramebufferTexture2D", &g.frameTex2D},
		{"glCheckFramebufferStatus", &g.checkFramebuffer},
		{"glDeleteFramebuffers", &g.delFramebuffers},
		{"glDeleteTextures", &g.deleteTextures},
		{"glCreateShader", &g.createShader},
		{"glShaderSource", &g.shaderSource},
		{"glCompileShader", &g.compileShader},
		{"glGetShaderiv", &g.getShaderiv},
		{"glGetShaderInfoLog", &g.shaderInfoLog},
		{"glCreateProgram", &g.createProgram},
		{"glAttachShader", &g.attachShader},
		{"glLinkProgram", &g.linkProgram},
		{"glGetProgramiv", &g.getProgramiv},
		{"glGetProgramInfoLog", &g.programInfoLog},
		{"glUseProgram", &g.useProgram},
		{"glDeleteShader", &g.deleteShader},
		{"glDeleteProgram", &g.deleteProgram},
		{"glGetUniformLocation", &g.uniformLocation},
		{"glUniform1i", &g.uniform1i},
		{"glUniform2f", &g.uniform2f},
		{"glEnableVertexAttribArray", &g.enableAttrib},
		{"glDisableVertexAttribArray", &g.disableAttrib},
		{"glVertexAttribPointer", &g.attribPointer},
		{"glVertexAttribDivisor", &g.attribDivisor},
		{"glDrawArrays", &g.drawArrays},
		{"glDrawArraysInstanced", &g.drawInstanced},
		{"glReadPixels", &g.readPixels},
		{"glFinish", &g.finish},
	}
	for _, item := range need {
		if err := bindProc(ctx, item.name, item.fn); err != nil {
			return nil, err
		}
	}
	_ = bindProc(ctx, "glUniform1ui", &g.uniform1ui)
	_ = bindProc(ctx, "glDispatchCompute", &g.dispatch)
	_ = bindProc(ctx, "glMemoryBarrier", &g.barrier)
	_ = bindProc(ctx, "glBindBufferBase", &g.bindBufferBase)
	_ = bindProc(ctx, "glMapBufferRange", &g.mapBuffer)
	_ = bindProc(ctx, "glUnmapBuffer", &g.unmapBuffer)
	_ = bindProc(ctx, "glGetIntegeri_v", &g.getIntegeri)
	g.readVersion(ctx.GLES())
	return g, nil
}

// glMaxComputeWorkGroupCount is the per-axis dispatch limit.
const glMaxComputeWorkGroupCount = 0x91BE

// workGroupLimit is how many work groups Dispatch may launch on each axis.
// The spec guarantees 65535. A larger phone frame does not fit on X alone.
func (g *glAPI) workGroupLimit() [3]uint32 {
	out := [3]uint32{65535, 65535, 65535}
	if g == nil || g.getIntegeri == nil {
		return out
	}
	if g.getError != nil {
		for g.getError() != 0 {
		}
	}
	for i := uint32(0); i < 3; i++ {
		var v int32
		g.getIntegeri(glMaxComputeWorkGroupCount, i, &v)
		if g.getError != nil && g.getError() != 0 {
			continue
		}
		if v > 0 {
			out[i] = uint32(v)
		}
	}
	return out
}

func bindProc(ctx glContext, name string, fn any) error {
	addr := ctx.Proc(name)
	if addr == 0 {
		return fmt.Errorf("%w: %s", ErrUnavailable, name)
	}
	native.Register(fn, addr)
	return nil
}

func (g *glAPI) readVersion(gles bool) {
	if g == nil || g.getIntegerv == nil {
		return
	}
	g.getIntegerv(glMajorVersion, &g.major)
	g.getIntegerv(glMinorVersion, &g.minor)
	g.renderer = native.GoString(g.getString(glRenderer))
	ver := native.GoString(g.getString(glVersion))
	g.gles = gles || strings.Contains(ver, "OpenGL ES")
	g.compute = g.dispatch != nil && g.bindBufferBase != nil && g.mapBuffer != nil && g.uniform1ui != nil
	if !g.compute {
		return
	}
	if g.gles {
		g.compute = g.major > 3 || (g.major == 3 && g.minor >= 1)
		return
	}
	g.compute = g.major > 4 || (g.major == 4 && g.minor >= 3)
}

func (g *glAPI) check(where string) error {
	if g == nil || g.getError == nil {
		return ErrClosed
	}
	code := g.getError()
	if code == 0 {
		return nil
	}
	for g.getError() != 0 {
	}
	return fmt.Errorf("%w: %s 0x%x", ErrUnavailable, where, code)
}

func (g *glAPI) source(shader uint32, src string) {
	buf := native.CString(src)
	ptr := unsafe.Pointer(&buf[0])
	g.shaderSource(shader, 1, uintptr(unsafe.Pointer(&ptr)), 0)
	runtime.KeepAlive(buf)
}

func (g *glAPI) log(shader bool, id uint32) string {
	var n int32
	if shader {
		g.getShaderiv(id, glInfoLogLength, &n)
	} else {
		g.getProgramiv(id, glInfoLogLength, &n)
	}
	if n < 2 {
		return ""
	}
	buf := make([]byte, n)
	if shader {
		g.shaderInfoLog(id, n, 0, uintptr(unsafe.Pointer(&buf[0])))
	} else {
		g.programInfoLog(id, n, 0, uintptr(unsafe.Pointer(&buf[0])))
	}
	runtime.KeepAlive(buf)
	return strings.TrimRight(string(buf), "\x00\n")
}

func (g *glAPI) compile(kind uint32, src string) (uint32, error) {
	sh := g.createShader(kind)
	if sh == 0 {
		return 0, fmt.Errorf("%w: glCreateShader", ErrUnavailable)
	}
	g.source(sh, src)
	g.compileShader(sh)
	var ok int32
	g.getShaderiv(sh, glCompileStatus, &ok)
	if ok == 0 {
		detail := g.log(true, sh)
		g.deleteShader(sh)
		if detail == "" {
			detail = "shader"
		}
		return 0, fmt.Errorf("%w: %s", ErrUnavailable, detail)
	}
	return sh, nil
}

func (g *glAPI) link(vert, frag uint32) (uint32, error) {
	prog := g.createProgram()
	if prog == 0 {
		return 0, fmt.Errorf("%w: glCreateProgram", ErrUnavailable)
	}
	g.attachShader(prog, vert)
	g.attachShader(prog, frag)
	g.linkProgram(prog)
	var ok int32
	g.getProgramiv(prog, glLinkStatus, &ok)
	g.deleteShader(vert)
	g.deleteShader(frag)
	if ok == 0 {
		detail := g.log(false, prog)
		g.deleteProgram(prog)
		if detail == "" {
			detail = "link"
		}
		return 0, fmt.Errorf("%w: %s", ErrUnavailable, detail)
	}
	return prog, nil
}

func (g *glAPI) uniform(prog uint32, name string) int32 {
	buf := native.CString(name)
	loc := g.uniformLocation(prog, uintptr(unsafe.Pointer(&buf[0])))
	runtime.KeepAlive(buf)
	return loc
}
