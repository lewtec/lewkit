//go:build !darwin && !ios

package opengl

import (
	"log/slog"
	"sync"
	"unsafe"
)

// Screen draws one GUI frame into a surface the caller owns.
type Screen struct {
	mu                  sync.Mutex
	run                 *runner
	ctx                 glContext
	gl                  *glAPI
	w, h                int
	nativeA             uintptr
	offscreen           bool
	closed              bool
	fbo, color          uint32
	vao, inst, tex      uint32
	fillProg, imageProg uint32
	fillExtent          int32
	fillSwap, imageSwap int32
	imageSampler        int32
}

func attach(ctx glContext, width, height int, offscreen bool, native uintptr) (*Screen, error) {
	if width < 1 || height < 1 || width > maxDim || height > maxDim {
		if ctx != nil {
			ctx.Destroy()
		}
		return nil, ErrSize
	}
	run, err := startRunner(ctx)
	if err != nil {
		return nil, err
	}
	s := &Screen{run: run, ctx: ctx, w: width, h: height, offscreen: offscreen, nativeA: native}
	if err := run.Do(func() error {
		api, err := loadAPI(ctx)
		if err != nil {
			return err
		}
		s.gl = api
		if err := s.initGL(); err != nil {
			return err
		}
		slog.Debug("opengl screen", "renderer", api.renderer, "major", api.major, "minor", api.minor, "gles", api.gles)
		return nil
	}); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Screen) initGL() error {
	g := s.gl
	g.disable(glDepthTest)
	g.disable(glCullFace)
	g.disable(glScissorTest)
	g.disable(glStencilTest)
	g.disable(glDither)
	g.disable(glMultisample)
	if g.getError != nil {
		for g.getError() != 0 {
		}
	}
	g.blendFunc(glOne, glOneMinusSrcAlpha)
	g.genVertexArrays(1, &s.vao)
	g.bindVertexArray(s.vao)
	vert, err := g.compile(glVertexShader, fillVertex(g))
	if err != nil {
		return err
	}
	frag, err := g.compile(glFragmentShader, fillFragment(g))
	if err != nil {
		g.deleteShader(vert)
		return err
	}
	s.fillProg, err = g.link(vert, frag)
	if err != nil {
		return err
	}
	vert, err = g.compile(glVertexShader, imageVertex(g))
	if err != nil {
		return err
	}
	frag, err = g.compile(glFragmentShader, imageFragment(g))
	if err != nil {
		g.deleteShader(vert)
		return err
	}
	s.imageProg, err = g.link(vert, frag)
	if err != nil {
		return err
	}
	s.fillExtent = g.uniform(s.fillProg, "uExtent")
	s.fillSwap = g.uniform(s.fillProg, "uSwapRB")
	s.imageSwap = g.uniform(s.imageProg, "uSwapRB")
	s.imageSampler = g.uniform(s.imageProg, "uPix")
	g.useProgram(s.imageProg)
	if s.imageSampler >= 0 {
		g.uniform1i(s.imageSampler, 0)
	}
	g.genBuffers(1, &s.inst)
	g.genTextures(1, &s.tex)
	g.bindTexture(glTexture2D, s.tex)
	g.texParameteri(glTexture2D, glTextureMinFilter, glNearest)
	g.texParameteri(glTexture2D, glTextureMagFilter, glNearest)
	g.texParameteri(glTexture2D, glTextureWrapS, glClampToEdge)
	g.texParameteri(glTexture2D, glTextureWrapT, glClampToEdge)
	if s.offscreen {
		if err := s.ensureTarget(); err != nil {
			return err
		}
	}
	return g.check("init")
}

func (s *Screen) ensureTarget() error {
	g := s.gl
	// A new texture name on every size. Mesa keeps the old attachment size
	// when glTexImage2D reallocates a texture that is already attached, so
	// a later larger draw only covers the previous rectangle.
	if s.fbo != 0 {
		g.bindFramebuffer(glFramebuffer, 0)
		g.delFramebuffers(1, &s.fbo)
		g.deleteTextures(1, &s.color)
		s.fbo, s.color = 0, 0
	}
	g.genFramebuffers(1, &s.fbo)
	g.genTextures(1, &s.color)
	g.bindTexture(glTexture2D, s.color)
	g.texParameteri(glTexture2D, glTextureMinFilter, glNearest)
	g.texParameteri(glTexture2D, glTextureMagFilter, glNearest)
	g.texImage2D(glTexture2D, 0, glRGBA8, int32(s.w), int32(s.h), 0, glRGBA, glUnsignedByte, 0)
	g.bindFramebuffer(glFramebuffer, s.fbo)
	g.frameTex2D(glFramebuffer, glColorAttachment0, glTexture2D, s.color, 0)
	g.bindTexture(glTexture2D, 0)
	if g.checkFramebuffer(glFramebuffer) != glFramebufferComplete {
		return ErrUnavailable
	}
	return g.check("framebuffer")
}

func (s *Screen) bindTarget() {
	if s.offscreen {
		s.gl.bindFramebuffer(glFramebuffer, s.fbo)
		return
	}
	s.gl.bindFramebuffer(glFramebuffer, 0)
}

// Draw paints under, then rounded rects, then glyph ink, and presents.
func (s *Screen) Draw(instances, under, ink []byte, width, height int) error {
	if s == nil {
		return ErrClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.run == nil {
		return ErrClosed
	}
	if width < 1 || height < 1 || width > maxDim || height > maxDim {
		return ErrSize
	}
	if len(instances)%instanceStride != 0 {
		return ErrSize
	}
	return s.run.Do(func() error {
		if width != s.w || height != s.h {
			s.w, s.h = width, height
			if s.offscreen {
				if err := s.ensureTarget(); err != nil {
					return err
				}
			}
		}
		return s.paint(instances, under, ink, width, height)
	})
}

func (s *Screen) paint(instances, under, ink []byte, width, height int) error {
	g := s.gl
	s.bindTarget()
	g.viewport(0, 0, int32(width), int32(height))
	g.disable(glBlend)
	g.clearColor(0, 0, 0, 1)
	g.clear(glColorBufferBit)
	g.enable(glBlend)
	need := width * height * 4
	if len(under) >= need {
		if err := s.blit(under[:need], width, height); err != nil {
			return err
		}
	}
	if fills := len(instances) / instanceStride; fills > 0 {
		if err := s.fills(instances, fills, width, height); err != nil {
			return err
		}
	}
	if len(ink) >= need {
		if err := s.blit(ink[:need], width, height); err != nil {
			return err
		}
	}
	if err := g.check("draw"); err != nil {
		return err
	}
	g.finish()
	if s.offscreen {
		return nil
	}
	return s.ctx.Swap()
}

func (s *Screen) fills(instances []byte, count, width, height int) error {
	g := s.gl
	g.bindVertexArray(s.vao)
	g.bindBuffer(glArrayBuffer, s.inst)
	g.bufferData(glArrayBuffer, len(instances), uintptr(unsafe.Pointer(&instances[0])), glDynamicDraw)
	stride := int32(instanceStride)
	for i := uint32(0); i < 4; i++ {
		g.enableAttrib(i)
		g.attribPointer(i, 4, glFloat, 0, stride, uintptr(i*16))
		g.attribDivisor(i, 1)
	}
	g.useProgram(s.fillProg)
	if s.fillExtent >= 0 {
		g.uniform2f(s.fillExtent, float32(width), float32(height))
	}
	if s.fillSwap >= 0 {
		g.uniform1i(s.fillSwap, 0)
	}
	g.drawInstanced(glTriangleStrip, 0, 4, int32(count))
	for i := uint32(0); i < 4; i++ {
		g.attribDivisor(i, 0)
		g.disableAttrib(i)
	}
	return nil
}

func (s *Screen) blit(pix []byte, width, height int) error {
	g := s.gl
	g.bindVertexArray(s.vao)
	g.activeTexture(glTexture0)
	g.bindTexture(glTexture2D, s.tex)
	g.pixelStorei(glUnpackAlignment, 1)
	g.texImage2D(glTexture2D, 0, glRGBA8, int32(width), int32(height), 0, glRGBA, glUnsignedByte, uintptr(unsafe.Pointer(&pix[0])))
	g.useProgram(s.imageProg)
	if s.imageSwap >= 0 {
		g.uniform1i(s.imageSwap, 0)
	}
	g.drawArrays(glTriangleStrip, 0, 4)
	return nil
}

// Read returns the RGBA8 frame, top to bottom. Offscreen screens keep it.
// A window screen reads the drawable back.
func (s *Screen) Read() ([]byte, error) {
	if s == nil {
		return nil, ErrClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.run == nil {
		return nil, ErrClosed
	}
	var out []byte
	err := s.run.Do(func() error {
		s.bindTarget()
		s.gl.pixelStorei(glPackAlignment, 1)
		raw := make([]byte, s.w*s.h*4)
		s.gl.readPixels(0, 0, int32(s.w), int32(s.h), glRGBA, glUnsignedByte, uintptr(unsafe.Pointer(&raw[0])))
		if err := s.gl.check("read"); err != nil {
			return err
		}
		flipRGBA(raw, s.w, s.h)
		out = raw
		return nil
	})
	return out, err
}

func flipRGBA(pix []byte, width, height int) {
	stride := width * 4
	tmp := make([]byte, stride)
	for y := 0; y < height/2; y++ {
		top := y * stride
		bot := (height - 1 - y) * stride
		copy(tmp, pix[top:top+stride])
		copy(pix[top:top+stride], pix[bot:bot+stride])
		copy(pix[bot:bot+stride], tmp)
	}
}

// Adopt points the screen at a replacement native handle and size.
// A zero handle reports [ErrLost]. X11 passes the display, which does not
// change when the window is resized; the size still updates.
func (s *Screen) Adopt(native uintptr, width, height int) error {
	if s == nil {
		return ErrClosed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.run == nil {
		return ErrClosed
	}
	if native == 0 && !s.offscreen {
		return ErrLost
	}
	if width < 1 || height < 1 || width > maxDim || height > maxDim {
		return ErrSize
	}
	return s.run.Do(func() error {
		s.w, s.h = width, height
		if s.offscreen {
			return s.ensureTarget()
		}
		if native != s.nativeA {
			if move, ok := s.ctx.(retargeter); ok {
				if err := move.Retarget(native, width, height); err != nil {
					return err
				}
				s.nativeA = native
			}
		}
		return nil
	})
}

// Close releases the context. A second call is a no-op.
func (s *Screen) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.run == nil {
		return nil
	}
	_ = s.run.Do(func() error {
		g := s.gl
		if g == nil {
			return nil
		}
		if s.fillProg != 0 {
			g.deleteProgram(s.fillProg)
		}
		if s.imageProg != 0 {
			g.deleteProgram(s.imageProg)
		}
		if s.vao != 0 {
			g.delVertexArrays(1, &s.vao)
		}
		if s.inst != 0 {
			g.deleteBuffers(1, &s.inst)
		}
		if s.tex != 0 {
			g.deleteTextures(1, &s.tex)
		}
		if s.color != 0 {
			g.deleteTextures(1, &s.color)
		}
		if s.fbo != 0 {
			g.delFramebuffers(1, &s.fbo)
		}
		return nil
	})
	s.run.Stop()
	s.run = nil
	return nil
}
