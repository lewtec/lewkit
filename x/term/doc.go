// Package term is a terminal multiplexer.
//
// The server is a database of one ubersession. Each pane is a record: the
// process, the grid, the size of the last client that wrote to it, and a
// slash-separated key. A folder is a directory the server stores, or the
// prefix of a pane key. A pane is a file in that tree. The server does not
// draw, and it does not store which client is showing which pane or
// standing in which folder. A client that reads those records differently
// does not restart the server. SessionInfo.level is the protocol level.
// A frame with the wrong version is rejected. A different level still
// attaches, and the desk warns. Level 2 stores directories. Level 3
// keeps the facts a pane stores for its session. Level 4 reads each
// pane's working directory from the process after its output stays quiet.
// Level 5 stores
// those facts on the pane id, so a move keeps them. A missing key
// inherits from the nearest ancestor directory. Directory facts move
// with the directory.
//
// The directory is read through x/driver/exec.Cwd after the pane's
// output stays quiet. Linux reads /proc and Darwin reads the process
// table. The next read waits for more output. The shell does not have
// to print the path. A program can also report it, and can store or
// drop a fact, with OSC.
// OSC 7 is file://host/path or an absolute path. OSC 1337 carries
// CurrentDir=. The lewkit sequences are ESC ] 777 ; cwd ; PATH,
// ESC ] 777 ; set ; KEY ; VALUE, and ESC ] 777 ; del ; KEY, ended by
// BEL or ST (ESC \). The bytes have no spaces. The drawer shows task,
// progress, and state. A value the pane writes is stored on that pane's
// id. A missing key is read from the nearest ancestor directory.
// Directory facts move and drop with the directory. Facts, including
// the last directory, are saved
// in term.state.json beside the socket and loaded when a new server
// starts. The pane processes are not brought back. The client's cwd is
// not stored, and a pane key does not create a host directory.
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
// status. + directory stores a folder on the server, including an empty
// one. The arrow on a session nests that session into a folder, and
// pane.move renames the keys. dir.move renames a stored directory. The
// desk shows the focused pane. Empty folders stay until they are removed.
// The client's cwd is not stored, and a pane key does not create a host
// directory. ctrl+k, and the same chord with super or meta, opens the
// session view. It does
// not search from the pane. Typing in that view filters sessions, panes,
// and actions. Folders stay open. One enter runs an action. The drawer takes clicks and the
// wheel. The wheel moves through the drawer the way the arrow keys do. A
// click or Enter selects a session, opens a folder, or arms a pane. The
// second Enter on that pane or session opens it. Right arrow, or the arrow
// on the row, picks up a pane, a session, or a folder. Enter or a click on
// a folder nests what is carried. / is the root. Up lands on it, and Enter
// or a click there unnests. Enter without a carry only selects or opens.
// A focused pane takes mouse, scroll, paste, and the other keys. The rail
// still takes a click that lands on it. Mouse bytes are written when the
// process asked for them. The wheel moves an alternate screen when it did
// not. A kitty image is drawn into the grid.
// The drawer is drawn with lipgloss. Bindings for individual actions are
// not assigned yet.
package term
