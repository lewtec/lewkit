// Package term is a terminal multiplexer.
//
// The server is a database of one ubersession. Each pane is a record: the
// process, the grid, and the size of the last client that wrote to it. The
// server does not draw, and it does not store which client is showing which
// pane. A client that reads those records differently does not restart the
// server.
//
// A client follows the panes it cares about. Following the ubersession means
// that client watches every pane. Another client can follow one pane. The
// choice stays on the client.
//
// A pane keeps the row and column count of the last client that wrote to it.
// A client that only reads does not resize the pane. The next write from a
// client whose screen size differs reflows the grid and the process.
//
// [Listen] serves the database on a unix socket. Each message is one varint
// size-delimited protobuf Envelope from package schema. [Connect] dials that
// socket and, when nothing is listening, starts the server the caller
// supplies. The server process stays up while a pane exists and exits when
// the last client disconnects and no panes remain. [Dial] and [Client.Call]
// are the RPC. [Run] is the attach view. It opens a command palette.
// Bindings for individual actions are not assigned yet; ctrl+p opens and
// closes the palette.
package term
