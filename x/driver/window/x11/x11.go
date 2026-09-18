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
	vals := []uint32{xproto.EventMaskExposure | xproto.EventMaskStructureNotify}
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
	wmDelete, err := setDeleteProtocol(conn, wid)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := xproto.MapWindowChecked(conn, wid).Check(); err != nil {
		conn.Close()
		return nil, err
	}

	win := &xwin{
		Buffer:   window.NewBuffer(w, h),
		conn:     conn,
		wid:      wid,
		gc:       gc,
		depth:    screen.RootDepth,
		wmDelete: wmDelete,
	}
	go win.loop()
	if ctx != nil {
		context.AfterFunc(ctx, func() { _ = win.Close() })
	}
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
	conn     *xgb.Conn
	wid      xproto.Window
	gc       xproto.Gcontext
	depth    byte
	wmDelete xproto.Atom
}

func (w *xwin) Draw() error {
	if err := w.Swap(); err != nil {
		return err
	}
	return w.put()
}

func (w *xwin) Resize(size image.Point) error {
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
	src := w.Front()
	bgra := make([]byte, len(src.Pix))
	window.ToBGRA(bgra, src)
	width, height := src.Rect.Dx(), src.Rect.Dy()
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
			uint16(width), uint16(n), 0, int16(y), 0, depth, bgra[y*stride:(y+n)*stride]).Check()
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
			_ = w.Buffer.Resize(image.Pt(int(e.Width), int(e.Height)))
		case xproto.ClientMessageEvent:
			if e.Type != 0 && w.wmDelete != 0 && e.Data.Data32[0] == uint32(w.wmDelete) {
				_ = w.Close()
				return
			}
		}
	}
}
