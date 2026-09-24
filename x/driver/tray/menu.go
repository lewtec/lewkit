package tray

// Node is one menu row in preorder. ID starts at 1. The root is not a Node.
type Node struct {
	ID       int
	Item     Item
	Children []Node
}

// Tree assigns stable ids. A click handler looks the id up with Find.
func Tree(items []Item) []Node {
	next := 1
	var walk func([]Item) []Node
	walk = func(items []Item) []Node {
		out := make([]Node, len(items))
		for i, item := range items {
			id := next
			next++
			out[i] = Node{ID: id, Item: item, Children: walk(item.Children)}
		}
		return out
	}
	return walk(items)
}

// Find returns the item with id.
func Find(nodes []Node, id int) (Item, bool) {
	for _, node := range nodes {
		if node.ID == id {
			return node.Item, true
		}
		if item, ok := Find(node.Children, id); ok {
			return item, true
		}
	}
	return Item{}, false
}
