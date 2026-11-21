package x11

import (
	"context"
	"image"
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_x11" }
func (factory) Name() string { return "X11" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return driver.RequireEnv(ctx, "DISPLAY")
}

func (factory) New(context.Context) (window.Driver, error) {
	return xdriver{}, nil
}

type xdriver struct{}

func (xdriver) Open(ctx context.Context, cfg window.Config) (window.Window, error) {
	w, h, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	screen := xproto.Setup(conn).DefaultScreen(conn)
	wid, err := xproto.NewWindowId(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	mask := uint32(xproto.CwEventMask)
	vals := []uint32{xproto.EventMaskExposure | xproto.EventMaskStructureNotify |
		xproto.EventMaskPointerMotion | xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease |
		xproto.EventMaskKeyPress | xproto.EventMaskKeyRelease | xproto.EventMaskButtonMotion}
	err = xproto.CreateWindowChecked(conn, screen.RootDepth, wid, screen.Root,
		0, 0, uint16(w), uint16(h), 0,
		xproto.WindowClassInputOutput, screen.RootVisual, mask, vals).Check()
	if err != nil {
		conn.Close()
		return nil, err
	}
	gc, err := xproto.NewGcontextId(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := xproto.CreateGCChecked(conn, gc, xproto.Drawable(wid), 0, nil).Check(); err != nil {
		conn.Close()
		return nil, err
	}
	if err := setTitle(conn, wid, cfg.Title); err != nil {
		conn.Close()
		return nil, err
	}
	if err := setMinSize(conn, wid); err != nil {
		conn.Close()
		return nil, err
	}
	wmDelete, err := setDeleteProtocol(conn, wid)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := xproto.MapWindowChecked(conn, wid).Check(); err != nil {
		conn.Close()
		return nil, err
	}

	buf := window.NewBuffer(w, h)
	buf.SetFramePeriod(cfg.Period)
	win := &xwin{
		Buffer:   buf,
		conn:     conn,
		wid:      wid,
		gc:       gc,
		depth:    screen.RootDepth,
		wmDelete: wmDelete,
		want:     window.WantSize{Width: w, Height: h},
	}
	go win.loop()
	window.CloseWhenDone(ctx, win)
	return win, nil
}

func setTitle(conn *xgb.Conn, wid xproto.Window, title string) error {
	if title == "" {
		return nil
	}
	b := []byte(title)
	return xproto.ChangePropertyChecked(conn, xproto.PropModeReplace, wid,
		xproto.AtomWmName, xproto.AtomString, 8, uint32(len(b)), b).Check()
}

func setMinSize(conn *xgb.Conn, wid xproto.Window) error {
	name, err := xproto.InternAtom(conn, false, uint16(len("WM_NORMAL_HINTS")), "WM_NORMAL_HINTS").Reply()
	if err != nil {
		return err
	}
	typ, err := xproto.InternAtom(conn, false, uint16(len("WM_SIZE_HINTS")), "WM_SIZE_HINTS").Reply()
	if err != nil {
		return err
	}
	// ICCCM WM_SIZE_HINTS: 18 CARD32s. flags=PMinSize (1<<4), min 1×1.
	var hints [18]uint32
	hints[0] = 1 << 4
	hints[5] = 1
	hints[6] = 1
	raw := make([]byte, 18*4)
	for i, v := range hints {
		raw[i*4] = byte(v)
		raw[i*4+1] = byte(v >> 8)
		raw[i*4+2] = byte(v >> 16)
		raw[i*4+3] = byte(v >> 24)
	}
	return xproto.ChangePropertyChecked(conn, xproto.PropModeReplace, wid,
		name.Atom, typ.Atom, 32, 18, raw).Check()
}

func setDeleteProtocol(conn *xgb.Conn, wid xproto.Window) (xproto.Atom, error) {
	protocols, err := xproto.InternAtom(conn, false, uint16(len("WM_PROTOCOLS")), "WM_PROTOCOLS").Reply()
	if err != nil {
		return 0, err
	}
	del, err := xproto.InternAtom(conn, false, uint16(len("WM_DELETE_WINDOW")), "WM_DELETE_WINDOW").Reply()
	if err != nil {
		return 0, err
	}
	var atom [4]byte
	atom[0] = byte(del.Atom)
	atom[1] = byte(del.Atom >> 8)
	atom[2] = byte(del.Atom >> 16)
	atom[3] = byte(del.Atom >> 24)
	err = xproto.ChangePropertyChecked(conn, xproto.PropModeReplace, wid,
		protocols.Atom, xproto.AtomAtom, 32, 1, atom[:]).Check()
	return del.Atom, err
}

type xwin struct {
	*window.Buffer
	mu       sync.Mutex
	blit     sync.Mutex
	conn     *xgb.Conn
	wid      xproto.Window
	gc       xproto.Gcontext
	depth    byte
	wmDelete xproto.Atom
	bgra     []byte
	want     window.WantSize
}

func (w *xwin) Size() image.Point {
	return window.HostSize(w.Buffer, &w.mu, &w.want)
}

func (w *xwin) Frame() *image.RGBA {
	return window.HostFrame(w.Buffer, w.Size())
}

func (w *xwin) Draw() error {
	return window.SwapBlit(w.Buffer, w.put)
}

func (w *xwin) Resize(size image.Point) error {
	w.mu.Lock()
	w.want.Width, w.want.Height = size.X, size.Y
	w.mu.Unlock()
	if err := w.Buffer.Resize(size); err != nil {
		return err
	}
	w.mu.Lock()
	conn := w.conn
	wid := w.wid
	w.mu.Unlock()
	if conn == nil {
		return window.ErrClosed
	}
	return xproto.ConfigureWindowChecked(conn, wid,
		xproto.ConfigWindowWidth|xproto.ConfigWindowHeight,
		[]uint32{uint32(size.X), uint32(size.Y)}).Check()
}

func (w *xwin) Close() error {
	w.mu.Lock()
	conn := w.conn
	w.conn = nil
	w.mu.Unlock()
	_ = w.Buffer.Close()
	if conn != nil {
		conn.Close()
	}
	return nil
}

func (w *xwin) put() error {
	w.mu.Lock()
	conn := w.conn
	wid := w.wid
	gc := w.gc
	depth := w.depth
	w.mu.Unlock()
	if conn == nil {
		return window.ErrClosed
	}
	w.blit.Lock()
	defer w.blit.Unlock()
	var width, height int
	w.WithFront(func(src *image.RGBA) {
		width, height = src.Rect.Dx(), src.Rect.Dy()
		if cap(w.bgra) < len(src.Pix) {
			w.bgra = make([]byte, len(src.Pix))
		} else {
			w.bgra = w.bgra[:len(src.Pix)]
		}
		window.ToBGRA(w.bgra, src)
	})
	stride := width * 4
	if stride == 0 || height == 0 {
		return nil
	}
	// X11 max request is typically 262140 bytes including the header.
	maxRows := (1 << 16) / stride
	if maxRows < 1 {
		maxRows = 1
	}
	for y := 0; y < height; y += maxRows {
		n := min(maxRows, height-y)
		err := xproto.PutImageChecked(conn, xproto.ImageFormatZPixmap, xproto.Drawable(wid), gc,
			uint16(width), uint16(n), 0, int16(y), 0, depth, w.bgra[y*stride:(y+n)*stride]).Check()
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *xwin) loop() {
	for {
		w.mu.Lock()
		conn := w.conn
		w.mu.Unlock()
		if conn == nil {
			return
		}
		ev, err := conn.WaitForEvent()
		if ev == nil && err == nil {
			_ = w.Close()
			return
		}
		if err != nil {
			_ = w.Close()
			return
		}
		switch e := ev.(type) {
		case xproto.ExposeEvent:
			if e.Count == 0 {
				w.Emit(window.Expose{})
				_ = w.put()
			}
		case xproto.ConfigureNotifyEvent:
			window.SetWant(w.Buffer, &w.mu, &w.want, int(e.Width), int(e.Height))
		case xproto.MotionNotifyEvent:
			w.Emit(window.Pointer{Pos: image.Pt(int(e.EventX), int(e.EventY)), Buttons: x11Buttons(e.State)})
		case xproto.ButtonPressEvent:
			w.x11Button(int(e.EventX), int(e.EventY), int(e.Detail), true, e.State)
		case xproto.ButtonReleaseEvent:
			w.x11Button(int(e.EventX), int(e.EventY), int(e.Detail), false, e.State)
		case xproto.KeyPressEvent:
			w.Emit(window.Key{Code: uint32(e.Detail), Pressed: true, Mod: x11Mod(e.State)})
		case xproto.KeyReleaseEvent:
			w.Emit(window.Key{Code: uint32(e.Detail), Pressed: false, Mod: x11Mod(e.State)})
		case xproto.ClientMessageEvent:
			if e.Type != 0 && w.wmDelete != 0 && e.Data.Data32[0] == uint32(w.wmDelete) {
				_ = w.Close()
				return
			}
		}
	}
}

func (w *xwin) x11Button(x, y, detail int, pressed bool, state uint16) {
	pos := image.Pt(x, y)
	switch detail {
	case 4:
		w.Emit(window.Scroll{Pos: pos, Delta: image.Pt(0, -12)})
	case 5:
		w.Emit(window.Scroll{Pos: pos, Delta: image.Pt(0, 12)})
	case 6:
		w.Emit(window.Scroll{Pos: pos, Delta: image.Pt(-12, 0)})
	case 7:
		w.Emit(window.Scroll{Pos: pos, Delta: image.Pt(12, 0)})
	default:
		w.Emit(window.Pointer{Pos: pos, Button: detail, Pressed: pressed, Buttons: x11Buttons(state)})
	}
}

func x11Buttons(state uint16) int {
	var b int
	if state&(1<<8) != 0 {
		b |= window.ButtonLeft
	}
	if state&(1<<9) != 0 {
		b |= window.ButtonMiddle
	}
	if state&(1<<10) != 0 {
		b |= window.ButtonRight
	}
	return b
}

func x11Mod(state uint16) window.Modifier {
	var m window.Modifier
	if state&1 != 0 {
		m |= window.ModShift
	}
	if state&(1<<2) != 0 {
		m |= window.ModCtrl
	}
	if state&(1<<3) != 0 {
		m |= window.ModAlt
	}
	if state&(1<<6) != 0 {
		m |= window.ModSuper
	}
	return m
}
