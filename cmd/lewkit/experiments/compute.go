package experiments

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"math"
	"os"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/vulkan"
)

// Compute is `lewkit experiments compute`.
type Compute struct {
	shader   *cmd.StringArg  `help:"SPIR-V path; open a window when set"`
	width    cmd.IntArg[int] `long:"width" default:"640" help:"output width"`
	height   cmd.IntArg[int] `long:"height" default:"480" help:"output height"`
	local    cmd.IntArg[int] `long:"local" default:"8" help:"local size for workgroup count"`
	bindings cmd.IntArg[int] `long:"bindings" default:"1" help:"storage-buffer count (2 adds time params)"`
}

func (Compute) Description() string {
	return "dispatch a compute shader; window if a SPIR-V path is given"
}

func (c *Compute) Run(ctx context.Context) error {
	if c.shader == nil {
		return runSmoke(ctx)
	}
	return c.runWindow(ctx)
}

func runSmoke(ctx context.Context) error {
	d, err := vulkan.Open(ctx)
	if err != nil {
		return err
	}
	defer d.Close()
	buf, err := d.Buffer(4)
	if err != nil {
		return err
	}
	defer buf.Close()
	sh, err := d.Shader(vulkan.SmokeSPIRV(), 1)
	if err != nil {
		return err
	}
	defer sh.Close()
	if err := d.Run(sh, 1, 1, 1, buf); err != nil {
		return err
	}
	out := make([]byte, 4)
	if err := buf.Read(out); err != nil {
		return err
	}
	fmt.Printf("%s: smoke -> %d\n", d.Name(), binary.LittleEndian.Uint32(out))
	return nil
}

func (c *Compute) runWindow(ctx context.Context) error {
	spirv, err := os.ReadFile(c.shader.Value())
	if err != nil {
		return err
	}
	binds := c.bindings.Value()
	if binds < 1 {
		return errBindings
	}
	local := c.local.Value()
	if local < 1 {
		return errLocal
	}
	d, err := vulkan.Open(ctx)
	if err != nil {
		return err
	}
	defer d.Close()
	sh, err := d.Shader(spirv, binds)
	if err != nil {
		return err
	}
	defer sh.Close()
	win, err := window.Open(ctx, window.Config{
		Title:  c.shader.Value(),
		Width:  c.width.Value(),
		Height: c.height.Value(),
	})
	if err != nil {
		return err
	}
	defer win.Close()

	var (
		pix, params *vulkan.Buffer
		bw, bh      int
		raw         []byte
		frame       uint32
	)
	defer func() {
		if pix != nil {
			pix.Close()
		}
		if params != nil {
			params.Close()
		}
	}()

	return window.Animate(ctx, win, time.Second/60, func(dst *image.RGBA, elapsed time.Duration) error {
		w, h := dst.Rect.Dx(), dst.Rect.Dy()
		if w < 1 || h < 1 {
			return nil
		}
		if pix == nil || w != bw || h != bh {
			if pix != nil {
				pix.Close()
				pix = nil
			}
			pix, err = d.Buffer(w * h * 4)
			if err != nil {
				return err
			}
			if binds >= 2 && params == nil {
				params, err = d.Buffer(16)
				if err != nil {
					return err
				}
			}
			bw, bh = w, h
			raw = make([]byte, w*h*4)
		}
		if params != nil {
			var p [16]byte
			binary.LittleEndian.PutUint32(p[0:], uint32(w))
			binary.LittleEndian.PutUint32(p[4:], uint32(h))
			binary.LittleEndian.PutUint32(p[8:], math.Float32bits(float32(elapsed.Seconds())))
			binary.LittleEndian.PutUint32(p[12:], frame)
			if err := params.Write(p[:]); err != nil {
				return err
			}
		}
		bufs := []*vulkan.Buffer{pix}
		if params != nil {
			bufs = append(bufs, params)
		}
		gx := uint32((w + local - 1) / local)
		gy := uint32((h + local - 1) / local)
		if err := d.Run(sh, gx, gy, 1, bufs...); err != nil {
			return err
		}
		if err := pix.Read(raw); err != nil {
			return err
		}
		copyRGBA(dst, raw)
		frame++
		return nil
	})
}

var (
	errBindings = errors.New("bindings must be >= 1")
	errLocal    = errors.New("local must be >= 1")
)

func copyRGBA(dst *image.RGBA, src []byte) {
	w, h := dst.Rect.Dx(), dst.Rect.Dy()
	for y := range h {
		row := src[y*w*4 : (y+1)*w*4]
		copy(dst.Pix[y*dst.Stride:], row)
	}
}
