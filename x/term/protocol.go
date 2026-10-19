package term

import "strings"

// ProtocolVersion is the frame version on Envelope.
// A frame with a different version is rejected. Fields a peer does not
// know stay readable, so this number stays put when the level changes.
const ProtocolVersion = 1

// ProtocolLevel is the behavior this process speaks.
// SessionInfo.level carries it. Level 1 had no stored directories.
// Level 2 keeps directories on the server. Level 3 reports a working
// directory when the process writes, and keeps session facts.
// Level 4 reads that directory from the process on a timer, on Linux
// and Darwin, including when the process prints nothing.
// A different level still attaches, and the desk warns.
const ProtocolLevel = 4

// ColorPalette means FG or BG is an ANSI palette index.
// Zero is the default color. 1 through 8 are normal and 9 through 16 are bright.
const ColorPalette = 0

// ColorIndex means FGIndex or BGIndex is a 256-color index.
// Index 0 is black, so the mode is what distinguishes it from the default.
const ColorIndex = 1

// ColorRGB means FGRGB or BGRGB is packed 0xRRGGBB.
// A zero channel is black, so the mode is what distinguishes it from unset.
const ColorRGB = 2

// Input bits are the private modes a pane's process turned on.
// The client uses them to encode mouse, focus, and paste for that pane.
// Zero means the process did not ask, and a click is not written as mouse bytes.
const (
	InputMouse     uint32 = 1 << iota // DEC 1000
	InputMotion                       // DEC 1002, motion while a button is held
	InputAnyMotion                    // DEC 1003
	InputSGR                          // DEC 1006
	InputFocus                        // DEC 1004
	InputPaste                        // DEC 2004
	InputAltScroll                    // DEC 1007, wheel sends cursor keys
	InputAlt                          // alternate screen
)

// Entry is one key in the session tree.
// Dir is a folder. Otherwise Pane names the file. Path is the full key.
type Entry struct {
	Name string
	Path string
	Dir  bool
	Pane string
}

// Tree is the direct children of one folder.
// Path empty is the root. The server does not store which client is standing there.
type Tree struct {
	Path    string
	Entries []Entry
}

// SessionInfo is the ubersession membership.
// Panes is creation order. It is not a layout.
// Entries is the key tree. A folder is a stored directory or a pane prefix.
// Level is the protocol level the server speaks. Zero means level 1.
// Dirs are directories the server keeps, including empty ones.
// Facts are values a pane stored for a session. Path is that session key.
type SessionInfo struct {
	ID      string
	Panes   []string
	Entries []Entry
	Level   uint32
	Dirs    []string
	Facts   []Fact
}

// Fact is one value stored for a session.
// Path is the session key. Key and Value are what the pane wrote.
type Fact struct {
	Path  string
	Key   string
	Value string
}

// PaneInfo is the pane record without the grid.
// Exit is meaningful when Alive is false. A signalled process may exit -1.
type PaneInfo struct {
	ID         string
	Title      string
	Command    []string
	Dir        string
	Alive      bool
	Exit       int
	Rows       int
	Cols       int
	Revision   uint64
	LastWriter string
	CursorRow  int
	CursorCol  int
	Path       string
	Cwd        string
}

// Cell is one column of the pane grid.
// Rune is empty for a blank column. Wide marks the second column of a
// double-width rune. FG and BG are palette indexes when FGMode and BGMode
// are ColorPalette. FGIndex and BGIndex are used for ColorIndex.
// FGRGB and BGRGB are used for ColorRGB.
type Cell struct {
	Rune    string
	FG      int
	BG      int
	Bold    bool
	Wide    bool
	FGMode  int
	BGMode  int
	FGIndex int
	BGIndex int
	FGRGB   uint32
	BGRGB   uint32
	Reverse bool
}

// Pane is one pane record, including the visible grid in row-major order.
// Cells has Rows times Cols entries. Scrollback stays on the server so a
// later write can reflow it; a client draws Cells.
// Input is the private-mode bits from the process. See [InputMouse].
type Pane struct {
	PaneInfo
	Cells []Cell
	Input uint32
}

// Line returns one visible row with trailing blanks removed.
// Wide spacers are skipped. Color is ignored, so a client can draw the text
// without using the attributes.
func (p Pane) Line(row int) string {
	if row < 0 || p.Cols <= 0 || row >= p.Rows {
		return ""
	}
	start := row * p.Cols
	end := start + p.Cols
	if start < 0 || end > len(p.Cells) {
		return ""
	}
	return strings.TrimRight(cellsText(p.Cells[start:end]), " ")
}

func cellsText(cells []Cell) string {
	var b strings.Builder
	for _, cell := range cells {
		if cell.Wide {
			continue
		}
		if cell.Rune == "" {
			b.WriteByte(' ')
			continue
		}
		b.WriteString(cell.Rune)
	}
	return b.String()
}

// CreateRequest starts a process in a new pane.
// An empty Command uses the SHELL environment variable, or /bin/sh.
// Rows and Cols are the initial grid. They are not a write, so LastWriter
// stays empty until a client sends input.
// Path is the pane's key. Empty stores the file at its id.
// A trailing slash stores it inside that folder, named by the new id.
type CreateRequest struct {
	Command []string
	Dir     string
	Rows    int
	Cols    int
	Title   string
	Client  string
	Path    string
}

// WriteRequest sends terminal input to a pane.
// Rows and Cols are the screen size of the client that is writing.
// Both zero keeps the current size. One of them set without the other is an error.
// A positive size that differs from the pane reflows the grid before the bytes are written.
type WriteRequest struct {
	ID     string
	Data   string
	Rows   int
	Cols   int
	Client string
}

// WatchRequest replaces the panes this connection wants pushed.
// All watches the ubersession. Otherwise Panes is the exact set, which may be empty.
// Prefix watches every pane whose key is that folder or a file inside it.
// A folder watch also receives session records. An exact pane set does not.
type WatchRequest struct {
	All    bool
	Panes  []string
	Prefix string
}

// Event is one pushed record.
type Event struct {
	Notice  Notice
	Pane    *Pane
	Session *SessionInfo
}

// Failure is the error object on an RPC response.
type Failure struct {
	Code string
	Text string
}

// Error is the server sentence.
func (f *Failure) Error() string {
	if f == nil {
		return ""
	}
	return f.Text
}

// Unwrap returns the sentinel named by Code.
func (f *Failure) Unwrap() error {
	if f == nil {
		return nil
	}
	switch f.Code {
	case ErrPane.Error():
		return ErrPane
	case ErrID.Error():
		return ErrID
	case ErrVersion.Error():
		return ErrVersion
	case ErrProtocol.Error():
		return ErrProtocol
	case ErrSocket.Error():
		return ErrSocket
	case ErrRunning.Error():
		return ErrRunning
	case ErrSize.Error():
		return ErrSize
	case ErrCommand.Error():
		return ErrCommand
	case ErrPty.Error():
		return ErrPty
	case ErrTermClosed.Error():
		return ErrTermClosed
	case ErrMethod.Error():
		return ErrMethod
	case ErrPath.Error():
		return ErrPath
	default:
		return nil
	}
}
