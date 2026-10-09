//go:build linux && !android

package wayland

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"golang.org/x/sys/unix"
)

type conn struct {
	fd      int
	writeMu sync.Mutex
	mu      sync.Mutex
	next    uint32
	handler map[uint32]func(uint16, []byte)
	in      *stream
	dead    chan struct{}
	errOnce sync.Once
	err     error
}

func dial(path string) (*conn, error) {
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	if err := unix.Connect(fd, &unix.SockaddrUnix{Name: path}); err != nil {
		_ = unix.Close(fd)
		return nil, err
	}
	c := &conn{
		fd:      fd,
		next:    1,
		handler: map[uint32]func(uint16, []byte){1: nil},
		dead:    make(chan struct{}),
		in: &stream{
			fd:      fd,
			scratch: make([]byte, 8192),
			oob:     make([]byte, unix.CmsgSpace(256)),
		},
	}
	c.handler[1] = c.onDisplay
	go c.readLoop()
	return c, nil
}

func (c *conn) readLoop() {
	for {
		id, op, payload, err := c.in.next()
		if err != nil {
			c.fail(err)
			return
		}
		c.dispatch(id, op, payload)
		if c.gone() {
			return
		}
	}
}

func (c *conn) onDisplay(op uint16, payload []byte) {
	switch op {
	case 0:
		_, payload = takeU32(payload)
		_, payload = takeU32(payload)
		msg, _ := takeString(payload)
		if msg == "" {
			msg = "display error"
		}
		c.fail(errors.New("wayland: " + msg))
	case 1:
		id, _ := takeU32(payload)
		c.remove(id)
	}
}

func (c *conn) dispatch(id uint32, op uint16, payload []byte) {
	c.mu.Lock()
	fn := c.handler[id]
	c.mu.Unlock()
	if fn != nil {
		fn(op, payload)
	}
}

func (c *conn) alloc(fn func(uint16, []byte)) uint32 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.next++
	id := c.next
	if fn != nil {
		c.handler[id] = fn
	}
	return id
}

func (c *conn) set(id uint32, fn func(uint16, []byte)) {
	c.mu.Lock()
	c.handler[id] = fn
	c.mu.Unlock()
}

func (c *conn) remove(id uint32) {
	c.mu.Lock()
	delete(c.handler, id)
	c.mu.Unlock()
}

func (c *conn) send(id uint32, opcode uint16, body []byte, fds ...int) error {
	if len(body)%4 != 0 {
		return errors.New("wayland: unaligned message")
	}
	size := 8 + len(body)
	msg := make([]byte, size)
	binary.LittleEndian.PutUint32(msg[0:], id)
	binary.LittleEndian.PutUint32(msg[4:], uint32(size)<<16|uint32(opcode))
	copy(msg[8:], body)
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return writeMsg(c.fd, msg, fds)
}

func (c *conn) roundtrip(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id := c.alloc(nil)
	ready := make(chan struct{})
	c.mu.Lock()
	c.handler[id] = func(op uint16, _ []byte) {
		if op != 0 {
			return
		}
		select {
		case <-ready:
		default:
			close(ready)
		}
	}
	c.mu.Unlock()
	if err := c.send(1, 0, pack(id)); err != nil {
		return err
	}
	select {
	case <-ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.dead:
		return c.fatal()
	}
}

func (c *conn) takeFD() int {
	if c.in == nil || len(c.in.fds) == 0 {
		return -1
	}
	fd := c.in.fds[0]
	c.in.fds = c.in.fds[1:]
	return fd
}

func (c *conn) fail(err error) {
	c.errOnce.Do(func() {
		c.err = err
		close(c.dead)
		_ = unix.Shutdown(c.fd, unix.SHUT_RDWR)
	})
}

func (c *conn) close() {
	c.fail(io.EOF)
	_ = unix.Close(c.fd)
}

func (c *conn) gone() bool {
	select {
	case <-c.dead:
		return true
	default:
		return false
	}
}

func (c *conn) fatal() error {
	if c.err != nil {
		return c.err
	}
	return errors.New("wayland: display closed")
}

type stream struct {
	fd      int
	buf     []byte
	fds     []int
	scratch []byte
	oob     []byte
}

func (s *stream) next() (uint32, uint16, []byte, error) {
	for {
		if len(s.buf) >= 8 {
			hdr := binary.LittleEndian.Uint32(s.buf[4:8])
			size := int(hdr >> 16)
			if size < 8 {
				return 0, 0, nil, errors.New("wayland: short header")
			}
			if len(s.buf) >= size {
				id := binary.LittleEndian.Uint32(s.buf[:4])
				op := uint16(hdr)
				payload := append([]byte(nil), s.buf[8:size]...)
				copy(s.buf, s.buf[size:])
				s.buf = s.buf[:len(s.buf)-size]
				return id, op, payload, nil
			}
		}
		n, oobn, _, _, err := unix.Recvmsg(s.fd, s.scratch, s.oob, 0)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return 0, 0, nil, err
		}
		if oobn > 0 {
			if err := s.rights(s.oob[:oobn]); err != nil {
				return 0, 0, nil, err
			}
		}
		if n == 0 {
			return 0, 0, nil, io.EOF
		}
		s.buf = append(s.buf, s.scratch[:n]...)
	}
}

func (s *stream) rights(oob []byte) error {
	msgs, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		fds, err := unix.ParseUnixRights(&msg)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			unix.CloseOnExec(fd)
			s.fds = append(s.fds, fd)
		}
	}
	return nil
}

func writeMsg(fd int, msg []byte, fds []int) error {
	var oob []byte
	if len(fds) > 0 {
		oob = unix.UnixRights(fds...)
	}
	for len(msg) > 0 || len(oob) > 0 {
		n, err := unix.SendmsgN(fd, msg, oob, nil, 0)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		oob = nil
		if n > len(msg) {
			n = len(msg)
		}
		msg = msg[n:]
		if n == 0 && len(msg) > 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func pack(args ...any) []byte {
	var b bytes.Buffer
	for _, arg := range args {
		switch v := arg.(type) {
		case uint32:
			var buf [4]byte
			binary.LittleEndian.PutUint32(buf[:], v)
			b.Write(buf[:])
		case int32:
			var buf [4]byte
			binary.LittleEndian.PutUint32(buf[:], uint32(v))
			b.Write(buf[:])
		case string:
			writeString(&b, v)
		default:
			panic("wayland pack")
		}
	}
	return b.Bytes()
}

func writeString(b *bytes.Buffer, s string) {
	raw := append([]byte(s), 0)
	var n [4]byte
	binary.LittleEndian.PutUint32(n[:], uint32(len(raw)))
	b.Write(n[:])
	b.Write(raw)
	if pad := (4 - len(raw)%4) % 4; pad != 0 {
		b.Write(make([]byte, pad))
	}
}

func takeU32(p []byte) (uint32, []byte) {
	if len(p) < 4 {
		return 0, nil
	}
	return binary.LittleEndian.Uint32(p), p[4:]
}

func takeI32(p []byte) (int32, []byte) {
	v, rest := takeU32(p)
	return int32(v), rest
}

func takeString(p []byte) (string, []byte) {
	if len(p) < 4 {
		return "", nil
	}
	n := int(binary.LittleEndian.Uint32(p))
	p = p[4:]
	if n < 1 || len(p) < n {
		return "", nil
	}
	s := string(p[:n-1])
	p = p[n:]
	if pad := (4 - n%4) % 4; pad > 0 {
		if len(p) < pad {
			return s, nil
		}
		p = p[pad:]
	}
	return s, p
}
