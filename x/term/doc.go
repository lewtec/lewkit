// Package term is a terminal multiplexer.
//
// The server is a database of one ubersession. Each pane is a record: the
// process, the grid, the size of the last client that wrote to it, and a
// slash-separated key. Folders are the prefixes of those keys. A pane is a
// file in that tree. The server does not draw, and it does not store which
// client is showing which pane or standing in which folder. A client that
// reads those records differently does not restart the server.
//
// A client follows the panes it cares about. Following the ubersession means
// that client watches every pane. Another client can follow one pane, or a
// folder. The choice stays on the client.
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
// are the RPC. [Run] is the attach desk. lew on the bottom rail opens the
// command palette, and so does ctrl+p. The palette stays closed until one
// of those asks for it. That rail keeps the recent panes.
// drawer lists sessions as a tree, and the x on a session removes those
// records. Each session row names that session's directory and its git
// status. + directory makes a pocket on this client. The arrow on a session
// nests that session into a folder, and pane.move renames the keys. The
// desk shows the focused pane. Empty pockets stay on the client. ctrl+k,
// and the same chord with super or meta, opens the session view. It does
// not search from the pane. Typing in that view filters sessions, panes,
// and actions. The second enter on a session or a pane leaves the session
// view focused on that pane. One enter runs an action.
// The drawer is drawn with lipgloss. Bindings for individual actions are
// not assigned yet.
package term
