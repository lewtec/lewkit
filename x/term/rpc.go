package term

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/lewtec/lewkit/x/errs"
)

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

type wireIn struct {
	V       int             `json:"v"`
	ID      int             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   *Failure        `json:"error"`
	Notice  string          `json:"notice"`
	Pane    *Pane           `json:"pane"`
	Session *SessionInfo    `json:"session"`
}

type wirePush struct {
	V       int          `json:"v"`
	Notice  Notice       `json:"notice"`
	Pane    *Pane        `json:"pane,omitempty"`
	Session *SessionInfo `json:"session,omitempty"`
}

type wireResponse struct {
	V      int      `json:"v"`
	ID     int      `json:"id"`
	Result any      `json:"result,omitempty"`
	Error  *Failure `json:"error,omitempty"`
}

type wireRequest struct {
	V      int    `json:"v"`
	ID     int    `json:"id"`
	Method Method `json:"method"`
	Params any    `json:"params,omitempty"`
}

type connState struct {
	session *Session
	conn    net.Conn
	enc     *json.Encoder
	bw      *bufio.Writer
	wmu     sync.Mutex

	mu          sync.Mutex
	queue       []Event
	wake        chan struct{}
	cancelWatch func()
}

func serveConn(ctx context.Context, session *Session, raw net.Conn) {
	defer raw.Close()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	state := &connState{
		session: session,
		conn:    raw,
		bw:      bufio.NewWriter(raw),
		wake:    make(chan struct{}, 1),
	}
	state.enc = json.NewEncoder(state.bw)
	go state.writeLoop(ctx)
	defer func() {
		state.mu.Lock()
		cancelWatch := state.cancelWatch
		state.mu.Unlock()
		if cancelWatch != nil {
			cancelWatch()
		}
	}()
	dec := json.NewDecoder(bufio.NewReader(raw))
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		var msg wireIn
		if err := dec.Decode(&msg); err != nil {
			return
		}
		state.reply(ctx, msg)
	}
}

func (c *connState) reply(ctx context.Context, msg wireIn) {
	if msg.V != ProtocolVersion {
		_ = c.send(wireResponse{V: ProtocolVersion, ID: msg.ID, Error: failure(ErrVersion)})
		return
	}
	method, err := ParseMethod(msg.Method)
	if err != nil {
		_ = c.send(wireResponse{V: ProtocolVersion, ID: msg.ID, Error: failure(err)})
		return
	}
	result, err := c.dispatch(ctx, method, msg.Params)
	if err != nil {
		_ = c.send(wireResponse{V: ProtocolVersion, ID: msg.ID, Error: failure(err)})
		return
	}
	_ = c.send(wireResponse{V: ProtocolVersion, ID: msg.ID, Result: result})
}

func (c *connState) dispatch(ctx context.Context, method Method, params json.RawMessage) (any, error) {
	switch method {
	case MethodSessionGet:
		info, err := c.session.Info(ctx)
		if err != nil {
			return nil, err
		}
		return struct {
			Session SessionInfo `json:"session"`
		}{Session: info}, nil
	case MethodPaneList:
		panes, err := c.session.Panes(ctx)
		if err != nil {
			return nil, err
		}
		return struct {
			Panes []PaneInfo `json:"panes"`
		}{Panes: panes}, nil
	case MethodPaneGet:
		var req struct {
			ID string `json:"id"`
		}
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		pane, err := c.session.Pane(ctx, req.ID)
		if err != nil {
			return nil, err
		}
		return struct {
			Pane Pane `json:"pane"`
		}{Pane: pane}, nil
	case MethodPaneCreate:
		var req CreateRequest
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		pane, err := c.session.Create(ctx, req)
		if err != nil {
			return nil, err
		}
		return struct {
			Pane Pane `json:"pane"`
		}{Pane: pane}, nil
	case MethodPaneWrite:
		var req WriteRequest
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		if err := c.session.Write(ctx, req); err != nil {
			return nil, err
		}
		return struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	case MethodPaneKill:
		var req struct {
			ID string `json:"id"`
		}
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		if err := c.session.Kill(ctx, req.ID); err != nil {
			return nil, err
		}
		return struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	case MethodWatch:
		var req WatchRequest
		if err := decodeParams(params, &req); err != nil {
			return nil, err
		}
		if err := c.rewatch(req); err != nil {
			return nil, err
		}
		return struct {
			OK bool `json:"ok"`
		}{OK: true}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrMethod, method)
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
			_ = c.send(wirePush{V: ProtocolVersion, Notice: ev.Notice, Pane: ev.Pane, Session: ev.Session})
		}
	}
}

func (c *connState) send(v any) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if err := c.enc.Encode(v); err != nil {
		return err
	}
	return c.bw.Flush()
}

func decodeParams(raw json.RawMessage, dest any) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("%w: %w", ErrProtocol, err)
	}
	return nil
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
	enc  *json.Encoder
	bw   *bufio.Writer
	wmu  sync.Mutex

	mu    sync.Mutex
	next  int
	wait  map[int]chan wireIn
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
		wait: map[int]chan wireIn{},
		wake: make(chan struct{}, 1),
	}
	client.enc = json.NewEncoder(client.bw)
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

// Call sends one method and stores the JSON result in dest when dest is non-nil.
func (c *Client) Call(ctx context.Context, method Method, params any, dest any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !method.IsValid() {
		return fmt.Errorf("%w: %s", ErrMethod, method)
	}
	c.mu.Lock()
	c.next++
	id := c.next
	ch := make(chan wireIn, 1)
	c.wait[id] = ch
	c.mu.Unlock()
	if err := c.send(wireRequest{V: ProtocolVersion, ID: id, Method: method, Params: params}); err != nil {
		c.mu.Lock()
		delete(c.wait, id)
		c.mu.Unlock()
		return err
	}
	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.wait, id)
		c.mu.Unlock()
		return ctx.Err()
	case msg, ok := <-ch:
		if !ok {
			return ErrTermClosed
		}
		if msg.Error != nil {
			return msg.Error
		}
		if dest != nil && len(msg.Result) > 0 {
			if err := json.Unmarshal(msg.Result, dest); err != nil {
				return fmt.Errorf("%w: %w", ErrProtocol, err)
			}
		}
		return nil
	}
}

func (c *Client) read(conn net.Conn) {
	dec := json.NewDecoder(bufio.NewReader(conn))
	for {
		var msg wireIn
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				c.fail(ErrTermClosed)
				return
			}
			c.fail(fmt.Errorf("%w: %w", ErrProtocol, err))
			return
		}
		if msg.Notice != "" {
			notice, err := ParseNotice(msg.Notice)
			if err != nil {
				continue
			}
			c.enqueue(Event{Notice: notice, Pane: msg.Pane, Session: msg.Session})
			continue
		}
		if msg.ID == 0 {
			continue
		}
		c.mu.Lock()
		ch := c.wait[msg.ID]
		delete(c.wait, msg.ID)
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
	c.wait = map[int]chan wireIn{}
	c.mu.Unlock()
	for _, ch := range wait {
		close(ch)
	}
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *Client) send(v any) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	if err := c.enc.Encode(v); err != nil {
		return fmt.Errorf("%w: %w", ErrSocket, err)
	}
	if err := c.bw.Flush(); err != nil {
		return fmt.Errorf("%w: %w", ErrSocket, err)
	}
	return nil
}

// Session reads the ubersession membership.
func (c *Client) Session(ctx context.Context) (SessionInfo, error) {
	var result struct {
		Session SessionInfo `json:"session"`
	}
	if err := c.Call(ctx, MethodSessionGet, struct{}{}, &result); err != nil {
		return SessionInfo{}, err
	}
	return result.Session, nil
}

// Panes reads every pane header.
func (c *Client) Panes(ctx context.Context) ([]PaneInfo, error) {
	var result struct {
		Panes []PaneInfo `json:"panes"`
	}
	if err := c.Call(ctx, MethodPaneList, struct{}{}, &result); err != nil {
		return nil, err
	}
	return result.Panes, nil
}

// Pane reads one pane, including its grid.
func (c *Client) Pane(ctx context.Context, id string) (Pane, error) {
	var result struct {
		Pane Pane `json:"pane"`
	}
	if err := c.Call(ctx, MethodPaneGet, map[string]string{"id": id}, &result); err != nil {
		return Pane{}, err
	}
	return result.Pane, nil
}

// Create starts a pane.
func (c *Client) Create(ctx context.Context, req CreateRequest) (Pane, error) {
	var result struct {
		Pane Pane `json:"pane"`
	}
	if err := c.Call(ctx, MethodPaneCreate, req, &result); err != nil {
		return Pane{}, err
	}
	return result.Pane, nil
}

// Write sends input. A positive size reflows when it differs from the pane.
func (c *Client) Write(ctx context.Context, req WriteRequest) error {
	var result struct {
		OK bool `json:"ok"`
	}
	return c.Call(ctx, MethodPaneWrite, req, &result)
}

// Kill stops a pane process.
func (c *Client) Kill(ctx context.Context, id string) error {
	var result struct {
		OK bool `json:"ok"`
	}
	return c.Call(ctx, MethodPaneKill, map[string]string{"id": id}, &result)
}

// Watch replaces the panes whose records this client wants pushed.
func (c *Client) Watch(ctx context.Context, req WatchRequest) error {
	var result struct {
		OK bool `json:"ok"`
	}
	return c.Call(ctx, MethodWatch, req, &result)
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
