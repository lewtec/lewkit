package term

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/term/schema"
	"google.golang.org/protobuf/encoding/protodelim"
	"google.golang.org/protobuf/proto"
)

// maxFrame is the largest size-delimited envelope accepted on the socket.
// A full viewport of cells can exceed the protodelim default of 4 MiB.
const maxFrame = 32 << 20

var frameReader = protodelim.UnmarshalOptions{MaxSize: maxFrame}

// Serve accepts RPC connections until ctx is cancelled.
func (s *Session) Serve(ctx context.Context, listener net.Listener) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		listener.Close()
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("%w: %w", ErrSocket, err)
		}
		go serveConn(ctx, s, conn)
	}
}

type connState struct {
	session *Session
	conn    net.Conn
	bw      *bufio.Writer
	wmu     sync.Mutex

	mu          sync.Mutex
	queue       []Event
	wake        chan struct{}
	cancelWatch func()
}

func serveConn(ctx context.Context, session *Session, raw net.Conn) {
	defer raw.Close()
	session.clientArrived()
	defer session.clientLeft()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	state := &connState{
		session: session,
		conn:    raw,
		bw:      bufio.NewWriter(raw),
		wake:    make(chan struct{}, 1),
	}
	go state.writeLoop(ctx)
	defer func() {
		state.mu.Lock()
		cancelWatch := state.cancelWatch
		state.mu.Unlock()
		if cancelWatch != nil {
			cancelWatch()
		}
	}()
	reader := bufio.NewReader(raw)
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		msg, err := readFrame(reader)
		if err != nil {
			return
		}
		state.reply(ctx, msg)
	}
}

func (c *connState) reply(ctx context.Context, msg *schema.Envelope) {
	if msg.GetVersion() != ProtocolVersion {
		_ = c.send(errorFrame(msg.GetId(), ErrVersion))
		return
	}
	method, err := ParseMethod(msg.GetMethod())
	if err != nil {
		_ = c.send(errorFrame(msg.GetId(), err))
		return
	}
	payload, err := c.dispatch(ctx, method, msg)
	if err != nil {
		_ = c.send(errorFrame(msg.GetId(), err))
		return
	}
	payload.Version = ProtocolVersion
	payload.Id = msg.GetId()
	_ = c.send(payload)
}

func (c *connState) dispatch(ctx context.Context, method Method, msg *schema.Envelope) (*schema.Envelope, error) {
	switch method {
	case MethodSessionGet:
		if err := payloadMatches(msg, msg.GetPayload() == nil); err != nil {
			return nil, err
		}
		info, err := c.session.Info(ctx)
		if err != nil {
			return nil, err
		}
		return &schema.Envelope{Payload: &schema.Envelope_Session{Session: sessionToSchema(info)}}, nil
	case MethodPaneList:
		if err := payloadMatches(msg, msg.GetPayload() == nil); err != nil {
			return nil, err
		}
		panes, err := c.session.Panes(ctx)
		if err != nil {
			return nil, err
		}
		return &schema.Envelope{Payload: &schema.Envelope_PaneList{PaneList: paneListToSchema(panes)}}, nil
	case MethodPaneGet:
		if err := payloadMatches(msg, msg.GetPaneId() != nil); err != nil {
			return nil, err
		}
		pane, err := c.session.Pane(ctx, msg.GetPaneId().GetId())
		if err != nil {
			return nil, err
		}
		return &schema.Envelope{Payload: &schema.Envelope_Pane{Pane: paneToSchema(pane)}}, nil
	case MethodPaneCreate:
		if err := payloadMatches(msg, msg.GetCreate() != nil); err != nil {
			return nil, err
		}
		pane, err := c.session.Create(ctx, createFromSchema(msg.GetCreate()))
		if err != nil {
			return nil, err
		}
		return &schema.Envelope{Payload: &schema.Envelope_Pane{Pane: paneToSchema(pane)}}, nil
	case MethodPaneWrite:
		if err := payloadMatches(msg, msg.GetWrite() != nil); err != nil {
			return nil, err
		}
		if err := c.session.Write(ctx, writeFromSchema(msg.GetWrite())); err != nil {
			return nil, err
		}
		return ackFrame(), nil
	case MethodPaneKill:
		if err := payloadMatches(msg, msg.GetPaneId() != nil); err != nil {
			return nil, err
		}
		if err := c.session.Kill(ctx, msg.GetPaneId().GetId()); err != nil {
			return nil, err
		}
		return ackFrame(), nil
	case MethodWatch:
		if err := payloadMatches(msg, msg.GetWatch() != nil); err != nil {
			return nil, err
		}
		if err := c.rewatch(watchFromSchema(msg.GetWatch())); err != nil {
			return nil, err
		}
		return ackFrame(), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrMethod, method)
	}
}

func ackFrame() *schema.Envelope {
	return &schema.Envelope{Payload: &schema.Envelope_Ack{Ack: &schema.Ack{Ok: true}}}
}

func errorFrame(id uint32, err error) *schema.Envelope {
	f := failure(err)
	return &schema.Envelope{
		Version: ProtocolVersion,
		Id:      id,
		Payload: &schema.Envelope_Error{Error: &schema.Failure{Code: f.Code, Text: f.Text}},
	}
}

func (c *connState) rewatch(req WatchRequest) error {
	cancel, err := c.session.Watch(req.Panes, req.All, c.enqueue)
	if err != nil {
		return err
	}
	c.mu.Lock()
	old := c.cancelWatch
	c.cancelWatch = cancel
	c.mu.Unlock()
	if old != nil {
		old()
	}
	return nil
}

func (c *connState) enqueue(ev Event) {
	c.mu.Lock()
	c.queue = coalesce(c.queue, ev)
	c.mu.Unlock()
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *connState) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.wake:
			c.flush()
		}
	}
}

func (c *connState) flush() {
	for {
		c.mu.Lock()
		queue := c.queue
		c.queue = nil
		c.mu.Unlock()
		if len(queue) == 0 {
			return
		}
		for _, ev := range queue {
			_ = c.send(pushFrame(ev))
		}
	}
}

func pushFrame(ev Event) *schema.Envelope {
	msg := &schema.Envelope{Version: ProtocolVersion, Notice: string(ev.Notice)}
	if ev.Pane != nil {
		msg.Payload = &schema.Envelope_Pane{Pane: paneToSchema(*ev.Pane)}
	}
	if ev.Session != nil {
		msg.Payload = &schema.Envelope_Session{Session: sessionToSchema(*ev.Session)}
	}
	return msg
}

func (c *connState) send(msg *schema.Envelope) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return writeFrame(c.bw, msg)
}

func writeFrame(w *bufio.Writer, msg *schema.Envelope) error {
	if _, err := protodelim.MarshalTo(w, msg); err != nil {
		return err
	}
	return w.Flush()
}

func readFrame(r *bufio.Reader) (*schema.Envelope, error) {
	msg := &schema.Envelope{}
	if err := frameReader.UnmarshalFrom(r, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func failure(err error) *Failure {
	if err == nil {
		return nil
	}
	return &Failure{Code: failureCode(err), Text: err.Error()}
}

func failureCode(err error) string {
	switch {
	case errors.Is(err, ErrPane):
		return ErrPane.Error()
	case errors.Is(err, ErrID):
		return ErrID.Error()
	case errors.Is(err, ErrVersion):
		return ErrVersion.Error()
	case errors.Is(err, ErrProtocol):
		return ErrProtocol.Error()
	case errors.Is(err, ErrSocket):
		return ErrSocket.Error()
	case errors.Is(err, ErrRunning):
		return ErrRunning.Error()
	case errors.Is(err, ErrSize):
		return ErrSize.Error()
	case errors.Is(err, ErrCommand):
		return ErrCommand.Error()
	case errors.Is(err, ErrPty):
		return ErrPty.Error()
	case errors.Is(err, ErrTermClosed):
		return ErrTermClosed.Error()
	case errors.Is(err, ErrMethod):
		return ErrMethod.Error()
	case errors.Is(err, errs.ErrNotFound):
		return errs.ErrNotFound.Error()
	case errors.Is(err, errs.ErrClosed):
		return ErrTermClosed.Error()
	default:
		return errs.ErrInvalid.Error()
	}
}

func coalesce(queue []Event, ev Event) []Event {
	for i := range queue {
		if sameNotice(queue[i], ev) {
			queue[i] = ev
			return queue
		}
	}
	if len(queue) >= 64 {
		queue = queue[1:]
	}
	return append(queue, ev)
}

func sameNotice(a, b Event) bool {
	if a.Notice != b.Notice {
		return false
	}
	if a.Notice == NoticeSession {
		return true
	}
	if a.Pane == nil || b.Pane == nil {
		return false
	}
	return a.Pane.ID == b.Pane.ID
}

// Client calls a term server.
type Client struct {
	conn net.Conn
	bw   *bufio.Writer
	wmu  sync.Mutex

	mu    sync.Mutex
	next  uint32
	wait  map[uint32]chan *schema.Envelope
	queue []Event
	wake  chan struct{}
	err   error
}

// Dial connects to a listening ubersession.
func Dial(ctx context.Context, path string) (*Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if path == "" {
		return nil, fmt.Errorf("%w: %s", ErrSocket, "empty socket path")
	}
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSocket, err)
	}
	client := &Client{
		conn: conn,
		bw:   bufio.NewWriter(conn),
		wait: map[uint32]chan *schema.Envelope{},
		wake: make(chan struct{}, 1),
	}
	go client.read(conn)
	return client, nil
}

// Close closes the socket.
func (c *Client) Close() error {
	err := c.conn.Close()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSocket, err)
	}
	return nil
}

// Call sends one method and returns the response envelope.
// params is the request message, or nil when the method has no body.
// A failure payload is returned as an error.
func (c *Client) Call(ctx context.Context, method Method, params proto.Message) (*schema.Envelope, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !method.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrMethod, method)
	}
	c.mu.Lock()
	c.next++
	if c.next == 0 {
		c.next = 1
	}
	id := c.next
	ch := make(chan *schema.Envelope, 1)
	c.wait[id] = ch
	c.mu.Unlock()
	msg := &schema.Envelope{Version: ProtocolVersion, Id: id, Method: string(method)}
	if err := setPayload(msg, params); err != nil {
		c.forget(id)
		return nil, err
	}
	if err := c.send(msg); err != nil {
		c.forget(id)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.forget(id)
		return nil, ctx.Err()
	case resp, ok := <-ch:
		if !ok {
			return nil, ErrTermClosed
		}
		if fail := resp.GetError(); fail != nil {
			return nil, failureFromSchema(fail)
		}
		return resp, nil
	}
}

func (c *Client) forget(id uint32) {
	c.mu.Lock()
	delete(c.wait, id)
	c.mu.Unlock()
}

func (c *Client) read(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		msg, err := readFrame(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.fail(ErrTermClosed)
				return
			}
			c.fail(fmt.Errorf("%w: %w", ErrProtocol, err))
			return
		}
		if msg.GetNotice() != "" {
			notice, err := ParseNotice(msg.GetNotice())
			if err != nil {
				continue
			}
			ev := Event{Notice: notice}
			if pane := msg.GetPane(); pane != nil {
				record := paneFromSchema(pane)
				ev.Pane = &record
			}
			if info := msg.GetSession(); info != nil {
				record := sessionFromSchema(info)
				ev.Session = &record
			}
			c.enqueue(ev)
			continue
		}
		if msg.GetId() == 0 {
			continue
		}
		c.mu.Lock()
		ch := c.wait[msg.GetId()]
		delete(c.wait, msg.GetId())
		c.mu.Unlock()
		if ch != nil {
			ch <- msg
		}
	}
}

func (c *Client) enqueue(ev Event) {
	c.mu.Lock()
	c.queue = coalesce(c.queue, ev)
	c.mu.Unlock()
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *Client) fail(err error) {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return
	}
	if err == nil {
		err = ErrTermClosed
	}
	c.err = err
	wait := c.wait
	c.wait = map[uint32]chan *schema.Envelope{}
	c.mu.Unlock()
	for _, ch := range wait {
		close(ch)
	}
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *Client) send(msg *schema.Envelope) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if err := writeFrame(c.bw, msg); err != nil {
		return fmt.Errorf("%w: %w", ErrSocket, err)
	}
	return nil
}

// Session reads the ubersession membership.
func (c *Client) Session(ctx context.Context) (SessionInfo, error) {
	resp, err := c.Call(ctx, MethodSessionGet, nil)
	if err != nil {
		return SessionInfo{}, err
	}
	info := resp.GetSession()
	if info == nil {
		return SessionInfo{}, fmt.Errorf("%w: %s", ErrProtocol, "session")
	}
	return sessionFromSchema(info), nil
}

// Panes reads every pane header.
func (c *Client) Panes(ctx context.Context) ([]PaneInfo, error) {
	resp, err := c.Call(ctx, MethodPaneList, nil)
	if err != nil {
		return nil, err
	}
	list := resp.GetPaneList()
	if list == nil {
		return nil, fmt.Errorf("%w: %s", ErrProtocol, "panes")
	}
	return paneListFromSchema(list), nil
}

// Pane reads one pane, including its grid.
func (c *Client) Pane(ctx context.Context, id string) (Pane, error) {
	resp, err := c.Call(ctx, MethodPaneGet, &schema.PaneId{Id: id})
	if err != nil {
		return Pane{}, err
	}
	pane := resp.GetPane()
	if pane == nil {
		return Pane{}, fmt.Errorf("%w: %s", ErrProtocol, "pane")
	}
	return paneFromSchema(pane), nil
}

// Create starts a pane.
func (c *Client) Create(ctx context.Context, req CreateRequest) (Pane, error) {
	resp, err := c.Call(ctx, MethodPaneCreate, createToSchema(req))
	if err != nil {
		return Pane{}, err
	}
	pane := resp.GetPane()
	if pane == nil {
		return Pane{}, fmt.Errorf("%w: %s", ErrProtocol, "pane")
	}
	return paneFromSchema(pane), nil
}

// Write sends input. A positive size reflows when it differs from the pane.
func (c *Client) Write(ctx context.Context, req WriteRequest) error {
	_, err := c.Call(ctx, MethodPaneWrite, writeToSchema(req))
	return err
}

// Kill stops a pane process.
func (c *Client) Kill(ctx context.Context, id string) error {
	_, err := c.Call(ctx, MethodPaneKill, &schema.PaneId{Id: id})
	return err
}

// Watch replaces the panes whose records this client wants pushed.
func (c *Client) Watch(ctx context.Context, req WatchRequest) error {
	_, err := c.Call(ctx, MethodWatch, watchToSchema(req))
	return err
}

// Next returns the next pushed record.
func (c *Client) Next(ctx context.Context) (Event, error) {
	for {
		if err := ctx.Err(); err != nil {
			return Event{}, err
		}
		c.mu.Lock()
		if len(c.queue) > 0 {
			ev := c.queue[0]
			c.queue = c.queue[1:]
			c.mu.Unlock()
			return ev, nil
		}
		err := c.err
		c.mu.Unlock()
		if err != nil {
			return Event{}, err
		}
		select {
		case <-ctx.Done():
			return Event{}, ctx.Err()
		case <-c.wake:
		}
	}
}
