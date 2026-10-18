package term

import "github.com/lewtec/lewkit/x/errs"

var (
	// ErrPane means that pane id is not in the ubersession.
	ErrPane = errs.Wrap(errs.ErrNotFound, "pane not found")
	// ErrID means a pane id was empty.
	ErrID = errs.Wrap(errs.ErrInvalid, "invalid pane id")
	// ErrVersion means the message is not protocol version 1.
	ErrVersion = errs.Wrap(errs.ErrInvalid, "unsupported term protocol")
	// ErrProtocol means a message was not a size-delimited record from the schema.
	ErrProtocol = errs.Wrap(errs.ErrSyntax, "term protocol syntax")
	// ErrSocket means the unix socket could not be opened or reached.
	ErrSocket = errs.Wrap(errs.ErrUnavailable, "term socket unavailable")
	// ErrRunning means a server is already listening on that socket.
	ErrRunning = errs.Wrap(errs.ErrAlreadyExists, "term server already running")
	// ErrSize means a write named only one of rows and columns.
	ErrSize = errs.Wrap(errs.ErrInvalid, "invalid pane size")
	// ErrCommand means the pane process could not be started.
	ErrCommand = errs.Wrap(errs.ErrNotFound, "pane command not found")
	// ErrPty means this OS cannot open a pseudoterminal.
	ErrPty = errs.Wrap(errs.ErrUnsupported, "pty is unsupported")
	// ErrTermClosed means the session or the pane is closed.
	ErrTermClosed = errs.Wrap(errs.ErrClosed, "term closed")
)
