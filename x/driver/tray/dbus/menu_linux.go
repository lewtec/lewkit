//go:build linux

package dbus

import (
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"

	"github.com/lewtec/lewkit/x/driver/tray"
)

type menuNode struct {
	item     *statusItem
	revision uint32
}

type layoutNode struct {
	ID         int32
	Properties map[string]dbus.Variant
	Children   []dbus.Variant
}

type propertyGroup struct {
	ID         int32
	Properties map[string]dbus.Variant
}

type eventGroup struct {
	ID        int32
	EventID   string
	Data      dbus.Variant
	Timestamp uint32
}

func (menu *menuNode) GetLayout(parentID int32, recursionDepth int32, _ []string) (uint32, layoutNode, *dbus.Error) {
	cfg := menu.item.snapshot()
	menu.item.mu.Lock()
	revision := menu.revision
	menu.item.mu.Unlock()
	nodes := indexLayout(tray.Tree(cfg.Menu))
	node, ok := nodes[parentID]
	if !ok {
		return revision, layoutNode{ID: parentID}, nil
	}
	if recursionDepth == 0 {
		node.Children = []dbus.Variant{}
	}
	return revision, node, nil
}

func (menu *menuNode) GetGroupProperties(ids []int32, _ []string) ([]propertyGroup, *dbus.Error) {
	cfg := menu.item.snapshot()
	nodes := indexLayout(tray.Tree(cfg.Menu))
	out := make([]propertyGroup, 0, len(ids))
	for _, id := range ids {
		node, ok := nodes[id]
		if !ok {
			continue
		}
		out = append(out, propertyGroup{ID: id, Properties: node.Properties})
	}
	return out, nil
}

func (menu *menuNode) GetProperty(id int32, name string) (dbus.Variant, *dbus.Error) {
	cfg := menu.item.snapshot()
	nodes := indexLayout(tray.Tree(cfg.Menu))
	node, ok := nodes[id]
	if !ok {
		return dbus.MakeVariant(""), nil
	}
	if value, ok := node.Properties[name]; ok {
		return value, nil
	}
	return dbus.MakeVariant(""), nil
}

func (menu *menuNode) Event(id int32, eventID string, data dbus.Variant, timestamp uint32) *dbus.Error {
	menu.handle(id, eventID, data, timestamp)
	return nil
}

func (menu *menuNode) EventGroup(events []eventGroup) ([]int32, *dbus.Error) {
	for _, event := range events {
		menu.handle(event.ID, event.EventID, event.Data, event.Timestamp)
	}
	return []int32{}, nil
}

func (menu *menuNode) AboutToShow(int32) (bool, *dbus.Error) {
	return false, nil
}

func (menu *menuNode) handle(id int32, eventID string, _ dbus.Variant, _ uint32) {
	if eventID != "clicked" && eventID != "activate" {
		return
	}
	cfg := menu.item.snapshot()
	item, ok := tray.Find(tray.Tree(cfg.Menu), int(id))
	if !ok || item.OnClick == nil || item.Separator || len(item.Children) > 0 {
		return
	}
	go item.OnClick()
}

func indexLayout(nodes []tray.Node) map[int32]layoutNode {
	out := map[int32]layoutNode{
		0: {
			ID:         0,
			Properties: map[string]dbus.Variant{"children-display": dbus.MakeVariant("submenu")},
			Children:   layoutChildren(nodes),
		},
	}
	var walk func([]tray.Node)
	walk = func(nodes []tray.Node) {
		for _, node := range nodes {
			out[int32(node.ID)] = layoutItem(node)
			walk(node.Children)
		}
	}
	walk(nodes)
	return out
}

func layoutChildren(nodes []tray.Node) []dbus.Variant {
	children := make([]dbus.Variant, 0, len(nodes))
	for _, node := range nodes {
		children = append(children, dbus.MakeVariant(layoutItem(node)))
	}
	return children
}

func layoutItem(node tray.Node) layoutNode {
	item := node.Item
	kind := "standard"
	if item.Separator {
		kind = "separator"
	}
	properties := map[string]dbus.Variant{
		"label":   dbus.MakeVariant(item.Label),
		"enabled": dbus.MakeVariant(!item.Disabled && !item.Separator),
		"visible": dbus.MakeVariant(true),
		"type":    dbus.MakeVariant(kind),
	}
	if item.Checked {
		properties["toggle-type"] = dbus.MakeVariant("checkmark")
		properties["toggle-state"] = dbus.MakeVariant(int32(1))
	}
	children := []dbus.Variant{}
	if len(node.Children) > 0 {
		properties["children-display"] = dbus.MakeVariant("submenu")
		children = layoutChildren(node.Children)
	}
	return layoutNode{ID: int32(node.ID), Properties: properties, Children: children}
}

func notifierSpec() introspect.Interface {
	return introspect.Interface{
		Name: itemInterface,
		Methods: []introspect.Method{
			{Name: "ContextMenu", Args: []introspect.Arg{{Name: "x", Type: "i", Direction: "in"}, {Name: "y", Type: "i", Direction: "in"}}},
			{Name: "Activate", Args: []introspect.Arg{{Name: "x", Type: "i", Direction: "in"}, {Name: "y", Type: "i", Direction: "in"}}},
			{Name: "SecondaryActivate", Args: []introspect.Arg{{Name: "x", Type: "i", Direction: "in"}, {Name: "y", Type: "i", Direction: "in"}}},
			{Name: "Scroll", Args: []introspect.Arg{{Name: "delta", Type: "i", Direction: "in"}, {Name: "orientation", Type: "s", Direction: "in"}}},
		},
		Properties: []introspect.Property{
			{Name: "Category", Type: "s", Access: "read"},
			{Name: "Id", Type: "s", Access: "read"},
			{Name: "Title", Type: "s", Access: "read"},
			{Name: "Status", Type: "s", Access: "read"},
			{Name: "WindowId", Type: "i", Access: "read"},
			{Name: "IconThemePath", Type: "s", Access: "read"},
			{Name: "Menu", Type: "o", Access: "read"},
			{Name: "ItemIsMenu", Type: "b", Access: "read"},
			{Name: "IconName", Type: "s", Access: "read"},
			{Name: "IconPixmap", Type: "a(iiay)", Access: "read"},
			{Name: "OverlayIconName", Type: "s", Access: "read"},
			{Name: "OverlayIconPixmap", Type: "a(iiay)", Access: "read"},
			{Name: "AttentionIconName", Type: "s", Access: "read"},
			{Name: "AttentionIconPixmap", Type: "a(iiay)", Access: "read"},
			{Name: "ToolTip", Type: "(sa(iiay)ss)", Access: "read"},
		},
		Signals: []introspect.Signal{
			{Name: "NewTitle"},
			{Name: "NewIcon"},
			{Name: "NewAttentionIcon"},
			{Name: "NewOverlayIcon"},
			{Name: "NewToolTip"},
			{Name: "NewStatus", Args: []introspect.Arg{{Name: "status", Type: "s"}}},
			{Name: "NewMenu"},
		},
	}
}

func menuSpec() introspect.Interface {
	return introspect.Interface{
		Name: menuInterface,
		Methods: []introspect.Method{
			{Name: "GetLayout", Args: []introspect.Arg{
				{Name: "parentId", Type: "i", Direction: "in"},
				{Name: "recursionDepth", Type: "i", Direction: "in"},
				{Name: "propertyNames", Type: "as", Direction: "in"},
				{Name: "revision", Type: "u", Direction: "out"},
				{Name: "layout", Type: "(ia{sv}av)", Direction: "out"},
			}},
			{Name: "GetGroupProperties", Args: []introspect.Arg{
				{Name: "ids", Type: "ai", Direction: "in"},
				{Name: "propertyNames", Type: "as", Direction: "in"},
				{Name: "properties", Type: "a(ia{sv})", Direction: "out"},
			}},
			{Name: "GetProperty", Args: []introspect.Arg{
				{Name: "id", Type: "i", Direction: "in"},
				{Name: "name", Type: "s", Direction: "in"},
				{Name: "value", Type: "v", Direction: "out"},
			}},
			{Name: "Event", Args: []introspect.Arg{
				{Name: "id", Type: "i", Direction: "in"},
				{Name: "eventId", Type: "s", Direction: "in"},
				{Name: "data", Type: "v", Direction: "in"},
				{Name: "timestamp", Type: "u", Direction: "in"},
			}},
			{Name: "EventGroup", Args: []introspect.Arg{
				{Name: "events", Type: "a(isvu)", Direction: "in"},
				{Name: "idErrors", Type: "ai", Direction: "out"},
			}},
			{Name: "AboutToShow", Args: []introspect.Arg{
				{Name: "id", Type: "i", Direction: "in"},
				{Name: "needUpdate", Type: "b", Direction: "out"},
			}},
		},
		Properties: []introspect.Property{
			{Name: "Version", Type: "u", Access: "read"},
			{Name: "TextDirection", Type: "s", Access: "read"},
			{Name: "Status", Type: "s", Access: "read"},
			{Name: "IconThemePath", Type: "as", Access: "read"},
		},
		Signals: []introspect.Signal{
			{Name: "ItemsPropertiesUpdated", Args: []introspect.Arg{
				{Name: "updatedProps", Type: "a(ia{sv})"},
				{Name: "removedProps", Type: "a(ias)"},
			}},
			{Name: "LayoutUpdated", Args: []introspect.Arg{
				{Name: "revision", Type: "u"},
				{Name: "parent", Type: "i"},
			}},
		},
	}
}
