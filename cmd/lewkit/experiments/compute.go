package experiments

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"math"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	_ "github.com/lewtec/lewkit/x/driver/prelude"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ffi/vulkan"
	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Compute is `lewkit experiments compute`.
type Compute struct {
	shader   *cmd.StringArg  `help:"SPIR-V or GLSL path; example.comp if omitted"`
	smoke    cmd.Flag        `long:"smoke" help:"print the 4-byte smoke shader instead of a window"`
	width    cmd.IntArg[int] `long:"width" default:"640" help:"output width"`
	height   cmd.IntArg[int] `long:"height" default:"480" help:"output height"`
	local    cmd.IntArg[int] `long:"local" default:"8" help:"local size for workgroup count"`
	bindings cmd.IntArg[int] `long:"bindings" default:"1" help:"storage-buffer count (2 adds time params)"`
}

func (Compute) Description() string {
	return "run a compute shader in a window (embedded GLSL example by default)"
}

func (c *Compute) Run(ctx context.Context) error {
	if c.smoke.Value() {
		return runSmoke(ctx)
	}
	return runDemo(ctx, c.runWindow)
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
	sh, err := d.Shader(ctx, vulkan.SmokeSPIRV(), 1)
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
	path := ""
	if c.shader != nil {
		path = c.shader.Value()
	}
	spirv, err := loadShader(ctx, path)
	if err != nil {
		return err
	}
	binds := c.bindings.Value()
	if path == "" {
		binds = 2
	}
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
	sh, err := d.Shader(ctx, spirv, binds)
	if err != nil {
		d.Close()
		return err
	}
	title := "lewkit compute"
	if path != "" {
		title = path
	}
	win, err := window.Open(ctx, window.Config{
		Title:  title,
		Width:  c.width.Value(),
		Height: c.height.Value(),
	})
	if err != nil {
		sh.Close()
		d.Close()
		return err
	}

	taskgroup.Go(ctx, "compute", taskgroup.CPU, func(ctx context.Context, st *taskgroup.Status) error {
		defer d.Close()
		defer sh.Close()
		defer win.Close()
		var (
			pix, params *vulkan.Buffer
			bw, bh      int
			raw         []byte
			frame       uint32
			last        time.Time
			fps         float64
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
			lewimage.CopyRGBA(dst, raw)
			now := time.Now()
			if !last.IsZero() {
				dt := now.Sub(last).Seconds()
				if dt > 0 {
					inst := 1 / dt
					if fps == 0 {
						fps = inst
					} else {
						fps = fps*0.85 + inst*0.15
					}
					st.Update(fmt.Sprintf("%.0f fps", fps))
				}
			}
			last = now
			frame++
			return nil
		})
	})
	return nil
}

var (
	errBindings = errors.New("bindings must be >= 1")
	errLocal    = errors.New("local must be >= 1")
)
