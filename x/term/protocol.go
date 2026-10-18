package term

import "strings"

// ProtocolVersion is the only version this server speaks.
// A different client can ignore fields it does not understand.
const ProtocolVersion = 1

// SessionInfo is the ubersession membership.
// Panes is creation order. It is not a layout.
type SessionInfo struct {
	ID    string
	Panes []string
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
}

// Cell is one column of the pane grid.
// Rune is empty for a blank column. Wide marks the second column of a
// double-width rune. FG and BG are palette indexes; 0 is the default color.
// 1 through 8 are the normal ANSI colors and 9 through 16 are the bright ones.
type Cell struct {
	Rune string
	FG   int
	BG   int
	Bold bool
	Wide bool
}

// Pane is one pane record, including the visible grid in row-major order.
// Cells has Rows times Cols entries. Scrollback stays on the server so a
// later write can reflow it; a client draws Cells.
type Pane struct {
	PaneInfo
	Cells []Cell
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
type CreateRequest struct {
	Command []string
	Dir     string
	Rows    int
	Cols    int
	Title   string
	Client  string
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
type WatchRequest struct {
	All   bool
	Panes []string
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
	default:
		return nil
	}
}
