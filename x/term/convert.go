package term

import (
	"fmt"

	"github.com/lewtec/lewkit/x/term/schema"
	"google.golang.org/protobuf/proto"
)

// Payload returns the message carried by env.
// It is nil when env has no payload.
func Payload(env *schema.Envelope) proto.Message {
	if env == nil {
		return nil
	}
	switch {
	case env.GetSession() != nil:
		return env.GetSession()
	case env.GetPaneList() != nil:
		return env.GetPaneList()
	case env.GetPane() != nil:
		return env.GetPane()
	case env.GetAck() != nil:
		return env.GetAck()
	case env.GetError() != nil:
		return env.GetError()
	case env.GetPaneId() != nil:
		return env.GetPaneId()
	case env.GetCreate() != nil:
		return env.GetCreate()
	case env.GetWrite() != nil:
		return env.GetWrite()
	case env.GetWatch() != nil:
		return env.GetWatch()
	default:
		return nil
	}
}

func setPayload(msg *schema.Envelope, params proto.Message) error {
	if params == nil {
		return nil
	}
	switch p := params.(type) {
	case *schema.SessionInfo:
		msg.Payload = &schema.Envelope_Session{Session: p}
	case *schema.PaneList:
		msg.Payload = &schema.Envelope_PaneList{PaneList: p}
	case *schema.Pane:
		msg.Payload = &schema.Envelope_Pane{Pane: p}
	case *schema.Ack:
		msg.Payload = &schema.Envelope_Ack{Ack: p}
	case *schema.Failure:
		msg.Payload = &schema.Envelope_Error{Error: p}
	case *schema.PaneId:
		msg.Payload = &schema.Envelope_PaneId{PaneId: p}
	case *schema.CreateRequest:
		msg.Payload = &schema.Envelope_Create{Create: p}
	case *schema.WriteRequest:
		msg.Payload = &schema.Envelope_Write{Write: p}
	case *schema.WatchRequest:
		msg.Payload = &schema.Envelope_Watch{Watch: p}
	default:
		return fmt.Errorf("%w: %T", ErrProtocol, params)
	}
	return nil
}

func payloadMatches(msg *schema.Envelope, match bool) error {
	if msg.GetPayload() != nil && !match {
		return fmt.Errorf("%w: %s", ErrProtocol, "payload")
	}
	return nil
}

func sessionToSchema(info SessionInfo) *schema.SessionInfo {
	return &schema.SessionInfo{Id: info.ID, Panes: copyStrings(info.Panes)}
}

func sessionFromSchema(info *schema.SessionInfo) SessionInfo {
	if info == nil {
		return SessionInfo{}
	}
	return SessionInfo{ID: info.GetId(), Panes: copyStrings(info.GetPanes())}
}

func paneInfoToSchema(info PaneInfo) *schema.PaneInfo {
	return &schema.PaneInfo{
		Id:         info.ID,
		Title:      info.Title,
		Command:    copyStrings(info.Command),
		Dir:        info.Dir,
		Alive:      info.Alive,
		Exit:       int32(info.Exit),
		Rows:       int32(info.Rows),
		Cols:       int32(info.Cols),
		Revision:   info.Revision,
		LastWriter: info.LastWriter,
		CursorRow:  int32(info.CursorRow),
		CursorCol:  int32(info.CursorCol),
	}
}

func paneInfoFromSchema(info *schema.PaneInfo) PaneInfo {
	if info == nil {
		return PaneInfo{}
	}
	return PaneInfo{
		ID:         info.GetId(),
		Title:      info.GetTitle(),
		Command:    copyStrings(info.GetCommand()),
		Dir:        info.GetDir(),
		Alive:      info.GetAlive(),
		Exit:       int(info.GetExit()),
		Rows:       int(info.GetRows()),
		Cols:       int(info.GetCols()),
		Revision:   info.GetRevision(),
		LastWriter: info.GetLastWriter(),
		CursorRow:  int(info.GetCursorRow()),
		CursorCol:  int(info.GetCursorCol()),
	}
}

func paneToSchema(pane Pane) *schema.Pane {
	info := paneInfoToSchema(pane.PaneInfo)
	return &schema.Pane{
		Id:         info.GetId(),
		Title:      info.GetTitle(),
		Command:    info.GetCommand(),
		Dir:        info.GetDir(),
		Alive:      info.GetAlive(),
		Exit:       info.GetExit(),
		Rows:       info.GetRows(),
		Cols:       info.GetCols(),
		Revision:   info.GetRevision(),
		LastWriter: info.GetLastWriter(),
		CursorRow:  info.GetCursorRow(),
		CursorCol:  info.GetCursorCol(),
		Cells:      cellsToSchema(pane.Cells),
	}
}

func paneFromSchema(pane *schema.Pane) Pane {
	if pane == nil {
		return Pane{}
	}
	return Pane{
		PaneInfo: paneInfoFromSchema(&schema.PaneInfo{
			Id:         pane.GetId(),
			Title:      pane.GetTitle(),
			Command:    pane.GetCommand(),
			Dir:        pane.GetDir(),
			Alive:      pane.GetAlive(),
			Exit:       pane.GetExit(),
			Rows:       pane.GetRows(),
			Cols:       pane.GetCols(),
			Revision:   pane.GetRevision(),
			LastWriter: pane.GetLastWriter(),
			CursorRow:  pane.GetCursorRow(),
			CursorCol:  pane.GetCursorCol(),
		}),
		Cells: cellsFromSchema(pane.GetCells()),
	}
}

func paneListToSchema(list []PaneInfo) *schema.PaneList {
	out := &schema.PaneList{Panes: make([]*schema.PaneInfo, len(list))}
	for i, info := range list {
		out.Panes[i] = paneInfoToSchema(info)
	}
	return out
}

func paneListFromSchema(list *schema.PaneList) []PaneInfo {
	if list == nil {
		return nil
	}
	in := list.GetPanes()
	out := make([]PaneInfo, len(in))
	for i, info := range in {
		out[i] = paneInfoFromSchema(info)
	}
	return out
}

func cellsToSchema(cells []Cell) []*schema.Cell {
	if cells == nil {
		return nil
	}
	out := make([]*schema.Cell, len(cells))
	for i, cell := range cells {
		out[i] = &schema.Cell{
			Rune: cell.Rune,
			Fg:   int32(cell.FG),
			Bg:   int32(cell.BG),
			Bold: cell.Bold,
			Wide: cell.Wide,
		}
	}
	return out
}

func cellsFromSchema(cells []*schema.Cell) []Cell {
	if cells == nil {
		return nil
	}
	out := make([]Cell, len(cells))
	for i, cell := range cells {
		if cell == nil {
			continue
		}
		out[i] = Cell{
			Rune: cell.GetRune(),
			FG:   int(cell.GetFg()),
			BG:   int(cell.GetBg()),
			Bold: cell.GetBold(),
			Wide: cell.GetWide(),
		}
	}
	return out
}

func createToSchema(req CreateRequest) *schema.CreateRequest {
	return &schema.CreateRequest{
		Command: copyStrings(req.Command),
		Dir:     req.Dir,
		Rows:    int32(req.Rows),
		Cols:    int32(req.Cols),
		Title:   req.Title,
		Client:  req.Client,
	}
}

func createFromSchema(req *schema.CreateRequest) CreateRequest {
	if req == nil {
		return CreateRequest{}
	}
	return CreateRequest{
		Command: copyStrings(req.GetCommand()),
		Dir:     req.GetDir(),
		Rows:    int(req.GetRows()),
		Cols:    int(req.GetCols()),
		Title:   req.GetTitle(),
		Client:  req.GetClient(),
	}
}

func writeToSchema(req WriteRequest) *schema.WriteRequest {
	return &schema.WriteRequest{
		Id:     req.ID,
		Data:   req.Data,
		Rows:   int32(req.Rows),
		Cols:   int32(req.Cols),
		Client: req.Client,
	}
}

func writeFromSchema(req *schema.WriteRequest) WriteRequest {
	if req == nil {
		return WriteRequest{}
	}
	return WriteRequest{
		ID:     req.GetId(),
		Data:   req.GetData(),
		Rows:   int(req.GetRows()),
		Cols:   int(req.GetCols()),
		Client: req.GetClient(),
	}
}

func watchToSchema(req WatchRequest) *schema.WatchRequest {
	return &schema.WatchRequest{All: req.All, Panes: copyStrings(req.Panes)}
}

func watchFromSchema(req *schema.WatchRequest) WatchRequest {
	if req == nil {
		return WatchRequest{}
	}
	return WatchRequest{All: req.GetAll(), Panes: copyStrings(req.GetPanes())}
}

func copyStrings(in []string) []string {
	if in == nil {
		return nil
	}
	return append([]string(nil), in...)
}

func failureFromSchema(f *schema.Failure) error {
	if f == nil {
		return nil
	}
	return &Failure{Code: f.GetCode(), Text: f.GetText()}
}
