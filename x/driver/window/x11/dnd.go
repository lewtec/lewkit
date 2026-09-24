package x11

import (
	"errors"
	"net/url"
	"strings"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/lewtec/lewkit/x/driver/window"
)

var errClientMessage = errors.New("client message")

type clientNote struct {
	dest xproto.Window
	typ  xproto.Atom
	data [5]uint32
}

type dndAtoms struct {
	aware, enter, position, status, leave, drop, finished xproto.Atom
	selection, typeList, uriList, actionCopy, property    xproto.Atom
}

func (w *xwin) enableDrop() {
	if w == nil || w.conn == nil {
		return
	}
	atoms, err := internDnd(w.conn)
	if err != nil {
		return
	}
	raw := []byte{5, 0, 0, 0}
	err = xproto.ChangePropertyChecked(w.conn, xproto.PropModeReplace, w.wid,
		atoms.aware, xproto.AtomAtom, 32, 1, raw).Check()
	if err != nil {
		return
	}
	w.dnd = atoms
}

func internDnd(conn *xgb.Conn) (*dndAtoms, error) {
	names := []string{
		"XdndAware", "XdndEnter", "XdndPosition", "XdndStatus", "XdndLeave",
		"XdndDrop", "XdndFinished", "XdndSelection", "XdndTypeList",
		"text/uri-list", "XdndActionCopy", "LEWKIT_DROP",
	}
	var atoms dndAtoms
	out := []*xproto.Atom{
		&atoms.aware, &atoms.enter, &atoms.position, &atoms.status, &atoms.leave,
		&atoms.drop, &atoms.finished, &atoms.selection, &atoms.typeList,
		&atoms.uriList, &atoms.actionCopy, &atoms.property,
	}
	for i, name := range names {
		reply, err := xproto.InternAtom(conn, false, uint16(len(name)), name).Reply()
		if err != nil {
			return nil, err
		}
		*out[i] = reply.Atom
	}
	return &atoms, nil
}

func (w *xwin) handleDnd(e xproto.ClientMessageEvent) bool {
	if w == nil || w.dnd == nil || len(e.Data.Data32) < 5 {
		return false
	}
	switch e.Type {
	case w.dnd.enter:
		w.dropSrc = xproto.Window(e.Data.Data32[0])
		w.dropOK = w.acceptsURI(e.Data.Data32)
		return true
	case w.dnd.position:
		w.dropSrc = xproto.Window(e.Data.Data32[0])
		action := xproto.Atom(0)
		flag := uint32(0)
		if w.dropOK {
			flag = 1
			action = w.dnd.actionCopy
		}
		if err := sendClient(w.conn, clientNote{
			dest: w.dropSrc, typ: w.dnd.status,
			data: [5]uint32{uint32(w.wid), flag, 0, 0, uint32(action)},
		}); err != nil {
			w.dropOK = false
		}
		return true
	case w.dnd.drop:
		w.dropSrc = xproto.Window(e.Data.Data32[0])
		if !w.dropOK {
			w.finishMessage(false)
			return true
		}
		if err := xproto.ConvertSelectionChecked(w.conn, w.wid, w.dnd.selection, w.dnd.uriList,
			w.dnd.property, xproto.TimeCurrentTime).Check(); err != nil {
			w.finishMessage(false)
		}
		return true
	case w.dnd.leave:
		w.dropSrc = 0
		w.dropOK = false
		return true
	default:
		return false
	}
}

func (w *xwin) acceptsURI(data []uint32) bool {
	if len(data) < 5 {
		return false
	}
	types := []xproto.Atom{xproto.Atom(data[2]), xproto.Atom(data[3]), xproto.Atom(data[4])}
	if data[1]&1 != 0 && w.dropSrc != 0 {
		reply, err := xproto.GetProperty(w.conn, false, w.dropSrc, w.dnd.typeList, xproto.AtomAtom, 0, 64).Reply()
		if err == nil && reply != nil {
			types = atomsFrom(reply.Value)
		}
	}
	for _, typ := range types {
		if typ == w.dnd.uriList {
			return true
		}
	}
	return false
}

func atomsFrom(raw []byte) []xproto.Atom {
	out := make([]xproto.Atom, 0, len(raw)/4)
	for i := 0; i+4 <= len(raw); i += 4 {
		out = append(out, xproto.Atom(uint32(raw[i])|uint32(raw[i+1])<<8|uint32(raw[i+2])<<16|uint32(raw[i+3])<<24))
	}
	return out
}

func (w *xwin) finishDrop(e xproto.SelectionNotifyEvent) {
	if w == nil || w.dnd == nil || e.Selection != w.dnd.selection {
		return
	}
	if e.Property == 0 {
		w.finishMessage(false)
		return
	}
	reply, err := xproto.GetProperty(w.conn, true, w.wid, e.Property, xproto.GetPropertyTypeAny, 0, 1<<20).Reply()
	if err != nil || reply == nil {
		w.finishMessage(false)
		return
	}
	paths := parseURIList(string(reply.Value))
	w.finishMessage(len(paths) > 0)
	if len(paths) > 0 {
		w.Emit(window.Drop{Paths: paths})
	}
}

func (w *xwin) finishMessage(ok bool) {
	if w == nil || w.dnd == nil || w.dropSrc == 0 || w.conn == nil {
		return
	}
	flag := uint32(0)
	action := xproto.Atom(0)
	if ok {
		flag = 1
		action = w.dnd.actionCopy
	}
	if err := sendClient(w.conn, clientNote{
		dest: w.dropSrc, typ: w.dnd.finished,
		data: [5]uint32{uint32(w.wid), flag, uint32(action), 0, 0},
	}); err != nil {
		w.dropSrc = 0
		w.dropOK = false
		return
	}
	w.dropSrc = 0
	w.dropOK = false
}

func sendClient(conn *xgb.Conn, note clientNote) error {
	if conn == nil {
		return errClientMessage
	}
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: note.dest,
		Type:   note.typ,
		Data:   xproto.ClientMessageDataUnionData32New(note.data[:]),
	}
	return xproto.SendEventChecked(conn, false, note.dest, xproto.EventMaskNoEvent, string(ev.Bytes())).Check()
}

func parseURIList(raw string) []string {
	var paths []string
	for line := range strings.SplitSeq(raw, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, "\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parsed, err := url.Parse(line)
		if err != nil || parsed.Scheme != "file" {
			continue
		}
		path, err := url.PathUnescape(parsed.Path)
		if err != nil || path == "" {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}
