package tview

import (
	"github.com/gdamore/tcell/v2"
)

// Tree navigation events.
const (
	treeNone int = iota
	treeHome
	treeEnd
	treeUp
	treeDown
	treePageUp
	treePageDown
	treeParent
	treeChild
	treeScrollUp // Move up without changing the selection, even when off screen.
	treeScrollDown
)

// TreeNode represents one node in a tree view.
type TreeNode struct {
	// The reference object.
	reference interface{}

	// This node's child nodes.
	children []*TreeNode

	// The item's text.
	text string

	// The text color.
	color tcell.Color

	// Whether or not this node can be selected.
	selectable bool

	// Whether or not this node's children should be displayed.
	expanded bool

	// The additional horizontal indent of this node's text.
	indent int

	// An optional function which is called when the user selects this node.
	selected func()

	// The hierarchy level (0 for the root, 1 for its children, and so on). This
	// is only up to date immediately after a call to process() (e.g. via
	// Draw()).
	level int

	// Temporary member variables.
	parent    *TreeNode // The parent node (nil for the root).
	graphicsX int       // The x-coordinate of the left-most graphics rune.
	textX     int       // The x-coordinate of the first rune of the text.
}

// NewTreeNode returns a new tree node.
func NewTreeNode(text string) *TreeNode {
	return &TreeNode{
		text:       text,
		color:      Styles.PrimaryTextColor,
		indent:     2,
		expanded:   true,
		selectable: true,
	}
}

// Walk traverses this node's subtree in depth-first, pre-order (NLR) order and
// calls the provided callback function on each traversed node (which includes
// this node) with the traversed node and its parent node (nil for this node).
// The callback returns whether traversal should continue with the traversed
// node's child nodes (true) or not recurse any deeper (false).
func (n *TreeNode) Walk(callback func(node, parent *TreeNode) bool) *TreeNode {
	n.parent = nil
	nodes := []*TreeNode{n}
	for len(nodes) > 0 {
		// Pop the top node and process it.
		node := nodes[len(nodes)-1]
		nodes = nodes[:len(nodes)-1]
		if !callback(node, node.parent) {
			// Don't add any children.
			continue
		}

		// Add children in reverse order.
		for index := len(node.children) - 1; index >= 0; index-- {
			node.children[index].parent = node
			nodes = append(nodes, node.children[index])
		}
	}

	return n
}

// SetReference allows you to store a reference of any type in this node. This
// will allow you to establish a mapping between the TreeView hierarchy and your
// internal tree structure.
func (n *TreeNode) SetReference(reference interface{}) *TreeNode {
	n.reference = reference
	return n
}

// GetReference returns this node's reference object.
func (n *TreeNode) GetReference() interface{} {
	return n.reference
}

// SetChildren sets this node's child nodes.
func (n *TreeNode) SetChildren(childNodes []*TreeNode) *TreeNode {
	n.children = childNodes
	return n
}

// GetText returns this node's text.
func (n *TreeNode) GetText() string {
	return n.text
}

// GetChildren returns this node's children.
func (n *TreeNode) GetChildren() []*TreeNode {
	return n.children
}

// ClearChildren removes all child nodes from this node.
func (n *TreeNode) ClearChildren() *TreeNode {
	n.children = nil
	return n
}

// AddChild adds a new child node to this node.
func (n *TreeNode) AddChild(node *TreeNode) *TreeNode {
	n.children = append(n.children, node)
	return n
}

// RemoveChild removes a child node from this node. If the child node cannot be
// found, nothing happens.
func (n *TreeNode) RemoveChild(node *TreeNode) *TreeNode {
	for index, child := range n.children {
		if child == node {
			n.children = append(n.children[:index], n.children[index+1:]...)
			break
		}
	}
	return n
}

// SetSelectable sets a flag indicating whether this node can be selected by
// the user.
func (n *TreeNode) SetSelectable(selectable bool) *TreeNode {
	n.selectable = selectable
	return n
}

// SetSelectedFunc sets a function which is called when the user selects this
// node by hitting Enter when it is selected.
func (n *TreeNode) SetSelectedFunc(handler func()) *TreeNode {
	n.selected = handler
	return n
}

// SetExpanded sets whether or not this node's child nodes should be displayed.
func (n *TreeNode) SetExpanded(expanded bool) *TreeNode {
	n.expanded = expanded
	return n
}

// Expand makes the child nodes of this node appear.
func (n *TreeNode) Expand() *TreeNode {
	n.expanded = true
	return n
}

// Collapse makes the child nodes of this node disappear.
func (n *TreeNode) Collapse() *TreeNode {
	n.expanded = false
	return n
}

// ExpandAll expands this node and all descendent nodes.
func (n *TreeNode) ExpandAll() *TreeNode {
	n.Walk(func(node, parent *TreeNode) bool {
		node.expanded = true
		return true
	})
	return n
}

// CollapseAll collapses this node and all descendent nodes.
func (n *TreeNode) CollapseAll() *TreeNode {
	n.Walk(func(node, parent *TreeNode) bool {
		n.expanded = false
		return true
	})
	return n
}

// IsExpanded returns whether the child nodes of this node are visible.
func (n *TreeNode) IsExpanded() bool {
	return n.expanded
}

// SetText sets the node's text which is displayed.
func (n *TreeNode) SetText(text string) *TreeNode {
	n.text = text
	return n
}

// GetColor returns the node's color.
func (n *TreeNode) GetColor() tcell.Color {
	return n.color
}

// SetColor sets the node's text color.
func (n *TreeNode) SetColor(color tcell.Color) *TreeNode {
	n.color = color
	return n
}

// SetIndent sets an additional indentation for this node's text. A value of 0
// keeps the text as far left as possible with a minimum of line graphics. Any
// value greater than that moves the text to the right.
func (n *TreeNode) SetIndent(indent int) *TreeNode {
	n.indent = indent
	return n
}

// GetLevel returns the node's level within the hierarchy, where 0 corresponds
// to the root node, 1 corresponds to its children, and so on. This is only
// guaranteed to be up to date immediately after the tree that contains this
// node is drawn.
func (n *TreeNode) GetLevel() int {
	return n.level
}

// TreeView displays tree structures. A tree consists of nodes (TreeNode
// objects) where each node has zero or more child nodes and exactly one parent
// node (except for the root node which has no parent node).
//
// The SetRoot() function is used to specify the root of the tree. Other nodes
// are added locally to the root node or any of its descendents. See the
// TreeNode documentation for details on node attributes. (You can use
// SetReference() to store a reference to nodes of your own tree structure.)
//
// Nodes can be selected by calling SetCurrentNode(). The user can navigate the
// selection or the tree by using the following keys:
//
//   - j, down arrow, right arrow: Move (the selection) down by one node.
//   - k, up arrow, left arrow: Move (the selection) up by one node.
//   - g, home: Move (the selection) to the top.
//   - G, end: Move (the selection) to the bottom.
//   - J: Move (the selection) up one level.
//   - K: Move (the selection) down one level (if it is shown).
//   - Ctrl-F, page down: Move (the selection) down by one page.
//   - Ctrl-B, page up: Move (the selection) up by one page.
//
// Selected nodes can trigger the "selected" callback when the user hits Enter.
//
// The root node corresponds to level 0, its children correspond to level 1,
// their children to level 2, and so on. Per default, the first level that is
// displayed is 0, i.e. the root node. You can call SetTopLevel() to hide
// levels.
//
// If graphics are turned on (see SetGraphics()), lines indicate the tree's
// hierarchy. Alternative (or additionally), you can set different prefixes
// using SetPrefixes() for different levels, for example to display hierarchical
// bullet point lists.
//
// See https://github.com/rivo/tview/wiki/TreeView for an example.
type TreeView struct {
	*Box

	// The root node.
	root *TreeNode

	// The currently selected node or nil if no node is selected.
	currentNode *TreeNode

	// The movement to be performed during the call to Draw(), one of the
	// constants defined above.
	movement int

	// The top hierarchical level shown. (0 corresponds to the root level.)
	topLevel int

	// Strings drawn before the nodes, based on their level.
	prefixes []string

	// Vertical scroll offset.
	offsetY int

	// If set to true, all node texts will be aligned horizontally.
	align bool

	// If set to true, the tree structure is drawn using lines.
	graphics bool

	// The color of the lines.
	graphicsColor tcell.Color

	// An optional function which is called when the user has navigated to a new
	// tree node.
	changed func(node *TreeNode)

	// An optional function which is called when a tree item was selected.
	selected func(node *TreeNode)

	// An optional function which is called when the user moves away from this
	// primitive.
	done func(key tcell.Key)

	// The visible nodes, top-down, as set by process().
	nodes []*TreeNode
}

// NewTreeView returns a new tree view.
func NewTreeView() *TreeView {
	return &TreeView{
		Box:           NewBox(),
		graphics:      true,
		graphicsColor: Styles.GraphicsColor,
	}
}

// SetRoot sets the root node of the tree.
func (t *TreeView) SetRoot(root *TreeNode) *TreeView {
	t.root = root
	return t
}

// GetRoot returns the root node of the tree. If no such node was previously
// set, nil is returned.
func (t *TreeView) GetRoot() *TreeNode {
	return t.root
}

// SetCurrentNode sets the currently selected node. Provide nil to clear all
// selections. Selected nodes must be visible and selectable, or else the
// selection will be changed to the top-most selectable and visible node.
//
// This function does NOT trigger the "changed" callback.
func (t *TreeView) SetCurrentNode(node *TreeNode) *TreeView {
	t.currentNode = node
	return t
}

// GetCurrentNode returns the currently selected node or nil of no node is
// currently selected.
func (t *TreeView) GetCurrentNode() *TreeNode {
	return t.currentNode
}

// SetTopLevel sets the first tree level that is visible with 0 referring to the
// root, 1 to the root's child nodes, and so on. Nodes above the top level are
// not displayed.
func (t *TreeView) SetTopLevel(topLevel int) *TreeView {
	t.topLevel = topLevel
	return t
}

// SetPrefixes defines the strings drawn before the nodes' texts. This is a
// slice of strings where each element corresponds to a node's hierarchy level,
// i.e. 0 for the root, 1 for the root's children, and so on (levels will
// cycle).
//
// For example, to display a hierarchical list with bullet points:
//
//   treeView.SetGraphics(false).
//     SetPrefixes([]string{"* ", "- ", "x "})
func (t *TreeView) SetPrefixes(prefixes []string) *TreeView {
	t.prefixes = prefixes
	return t
}

// SetAlign controls the horizontal alignment of the node texts. If set to true,
// all texts except that of top-level nodes will be placed in the same column.
// If set to false, they will indent with the hierarchy.
func (t *TreeView) SetAlign(align bool) *TreeView {
	t.align = align
	return t
}

// SetGraphics sets a flag which determines whether or not line graphics are
// drawn to illustrate the tree's hierarchy.
func (t *TreeView) SetGraphics(showGraphics bool) *TreeView {
	t.graphics = showGraphics
	return t
}

// SetGraphicsColor sets the colors of the lines used to draw the tree structure.
func (t *TreeView) SetGraphicsColor(color tcell.Color) *TreeView {
	t.graphicsColor = color
	return t
}

// SetChangedFunc sets the function which is called when the user navigates to
// a new tree node.
func (t *TreeView) SetChangedFunc(handler func(node *TreeNode)) *TreeView {
	t.changed = handler
	return t
}

// SetSelectedFunc sets the function which is called when the user selects a
// node by pressing Enter on the current selection.
func (t *TreeView) SetSelectedFunc(handler func(node *TreeNode)) *TreeView {
	t.selected = handler
	return t
}

// SetDoneFunc sets a handler which is called whenever the user presses the
// Escape, Tab, or Backtab key.
func (t *TreeView) SetDoneFunc(handler func(key tcell.Key)) *TreeView {
	t.done = handler
	return t
}

// GetScrollOffset returns the number of node rows that were skipped at the top
// of the tree view. Note that when the user navigates the tree view, this value
// is only updated after the tree view has been redrawn.
func (t *TreeView) GetScrollOffset() int {
	return t.offsetY
}

// GetRowCount returns the number of "visible" nodes. This includes nodes which
// fall outside the tree view's box but notably does not include the children
// of collapsed nodes. Note that this value is only up to date after the tree
// view has been drawn.
func (t *TreeView) GetRowCount() int {
	return len(t.nodes)
}

// process builds the visible tree, populates the "nodes" slice, and processes
// pending selection actions.
func (t *TreeView) process() {
	_, _, _, height := t.GetInnerRect()

	// Determine visible nodes and their placement.
	var graphicsOffset, maxTextX, parentSelectedIndex int
	t.nodes = nil
	if t.root == nil {
		return
	}
	selectedIndex := -1
	topLevelGraphicsX := -1
	if t.graphics {
		graphicsOffset = 1
	}
	t.root.Walk(func(node, parent *TreeNode) bool {
		// Set node attributes.
		node.parent = parent
		if parent == nil {
			node.level = 0
			node.graphicsX = 0
			node.textX = 0
		} else {
			node.level = parent.level + 1
			node.graphicsX = parent.textX
			node.textX = node.graphicsX + graphicsOffset + node.indent
		}
		if !t.graphics && t.align {
			// Without graphics, we align nodes on the first column.
			node.textX = 0
		}
		if node.level == t.topLevel {
			// No graphics for top level nodes.
			node.graphicsX = 0
			node.textX = 0
		}

		// Add the node to the list.
		if node.level >= t.topLevel {
			// This node will be visible.
			if node.textX > maxTextX {
				maxTextX = node.textX
			}
			if node == t.currentNode && node.selectable {
				selectedIndex = len(t.nodes)
			}

			// Maybe we want to skip this level.
			if t.topLevel == node.level && (topLevelGraphicsX < 0 || node.graphicsX < topLevelGraphicsX) {
				topLevelGraphicsX = node.graphicsX
			}

			t.nodes = append(t.nodes, node)
		}

		// Keep track of the parent of the selected node.
		if selectedIndex < 0 && node.selectable && len(node.children) > 0 && node.expanded {
			parentSelectedIndex = len(t.nodes) - 1
		}

		// Recurse if desired.
		return node.expanded
	})

	// Post-process positions.
	for _, node := range t.nodes {
		// If text must align, we correct the positions.
		if t.align && node.level > t.topLevel {
			node.textX = maxTextX
		}

		// If we skipped levels, shift to the left.
		if topLevelGraphicsX > 0 {
			node.graphicsX -= topLevelGraphicsX
			node.textX -= topLevelGraphicsX
		}
	}

	// Process selection. (Also trigger events if necessary.)
	if selectedIndex >= 0 {
		// Move the selection.
		newSelectedIndex := selectedIndex
	MovementSwitch:
		switch t.movement {
		case treeUp:
			for newSelectedIndex > 0 {
				newSelectedIndex--
				if t.nodes[newSelectedIndex].selectable {
					break MovementSwitch
				}
			}
			newSelectedIndex = selectedIndex
		case treeDown:
			for newSelectedIndex < len(t.nodes)-1 {
				newSelectedIndex++
				if t.nodes[newSelectedIndex].selectable {
					break MovementSwitch
				}
			}
			newSelectedIndex = selectedIndex
		case treeHome:
			for newSelectedIndex = 0; newSelectedIndex < len(t.nodes); newSelectedIndex++ {
				if t.nodes[newSelectedIndex].selectable {
					break MovementSwitch
				}
			}
			newSelectedIndex = selectedIndex
		case treeEnd:
			for newSelectedIndex = len(t.nodes) - 1; newSelectedIndex >= 0; newSelectedIndex-- {
				if t.nodes[newSelectedIndex].selectable {
					break MovementSwitch
				}
			}
			newSelectedIndex = selectedIndex
		case treePageDown:
			if newSelectedIndex+height < len(t.nodes) {
				newSelectedIndex += height
			} else {
				newSelectedIndex = len(t.nodes) - 1
			}
			for ; newSelectedIndex < len(t.nodes); newSelectedIndex++ {
				if t.nodes[newSelectedIndex].selectable {
					break MovementSwitch
				}
			}
			newSelectedIndex = selectedIndex
		case treePageUp:
			if newSelectedIndex >= height {
				newSelectedIndex -= height
			} else {
				newSelectedIndex = 0
			}
			for ; newSelectedIndex >= 0; newSelectedIndex-- {
				if t.nodes[newSelectedIndex].selectable {
					break MovementSwitch
				}
			}
			newSelectedIndex = selectedIndex
		case treeParent:
			newSelectedIndex = parentSelectedIndex
		case treeChild:
			for newSelectedIndex < len(t.nodes)-1 {
				newSelectedIndex++
				if t.nodes[newSelectedIndex].selectable && t.nodes[newSelectedIndex].parent == t.nodes[selectedIndex] {
					break MovementSwitch
				}
			}
			newSelectedIndex = selectedIndex
		}
		t.currentNode = t.nodes[newSelectedIndex]
		if newSelectedIndex != selectedIndex {
			t.movement = treeNone
			if t.changed != nil {
				t.changed(t.currentNode)
			}
		}
		selectedIndex = newSelectedIndex

		// Move selection into viewport.
		if t.movement != treeScrollDown && t.movement != treeScrollUp {
			if selectedIndex-t.offsetY >= height {
				t.offsetY = selectedIndex - height + 1
			}
			if selectedIndex < t.offsetY {
				t.offsetY = selectedIndex
			}
		}
	} else {
		// If selection is not visible or selectable, select the first candidate.
		if t.currentNode != nil {
			for index, node := range t.nodes {
				if node.selectable {
					selectedIndex = index
					t.currentNode = node
					break
				}
			}
		}
		if selectedIndex < 0 {
			t.currentNode = nil
		}
	}
}

// Draw draws this primitive onto the screen.
func (t *TreeView) Draw(screen tcell.Screen) {
	t.Box.DrawForSubclass(screen, t)
	if t.root == nil {
		return
	}
	_, totalHeight := screen.Size()

	t.process()

	// Scroll the tree.
	x, y, width, height := t.GetInnerRect()
	switch t.movement {
	case treeUp, treeScrollUp:
		t.offsetY--
	case treeDown, treeScrollDown:
		t.offsetY++
	case treeHome:
		t.offsetY = 0
	case treeEnd:
		t.offsetY = len(t.nodes)
	case treePageUp:
		t.offsetY -= height
	case treePageDown:
		t.offsetY += height
	}
	t.movement = treeNone

	// Fix invalid offsets.
	if t.offsetY >= len(t.nodes)-height {
		t.offsetY = len(t.nodes) - height
	}
	if t.offsetY < 0 {
		t.offsetY = 0
	}

	// Draw the tree.
	posY := y
	lineStyle := tcell.StyleDefault.Background(t.backgroundColor).Foreground(t.graphicsColor)
	for index, node := range t.nodes {
		// Skip invisible parts.
		if posY >= y+height+1 || posY >= totalHeight {
			break
		}
		if index < t.offsetY {
			continue
		}

		// Draw the graphics.
		if t.graphics {
			// Draw ancestor branches.
			ancestor := node.parent
			for ancestor != nil && ancestor.parent != nil && ancestor.parent.level >= t.topLevel {
				if ancestor.graphicsX >= width {
					continue
				}

				// Draw a branch if this ancestor is not a last child.
				if ancestor.parent.children[len(ancestor.parent.children)-1] != ancestor {
					if posY-1 >= y && ancestor.textX > ancestor.graphicsX {
						PrintJoinedSemigraphics(screen, x+ancestor.graphicsX, posY-1, Borders.Vertical, lineStyle)
					}
					if posY < y+height {
						screen.SetContent(x+ancestor.graphicsX, posY, Borders.Vertical, nil, lineStyle)
					}
				}
				ancestor = ancestor.parent
			}

			if node.textX > node.graphicsX && node.graphicsX < width {
				// Connect to the node above.
				if posY-1 >= y && t.nodes[index-1].graphicsX <= node.graphicsX && t.nodes[index-1].textX > node.graphicsX {
					PrintJoinedSemigraphics(screen, x+node.graphicsX, posY-1, Borders.TopLeft, lineStyle)
				}

				// Join this node.
				if posY < y+height {
					screen.SetContent(x+node.graphicsX, posY, Borders.BottomLeft, nil, lineStyle)
					for pos := node.graphicsX + 1; pos < node.textX && pos < width; pos++ {
						screen.SetContent(x+pos, posY, Borders.Horizontal, nil, lineStyle)
					}
				}
			}
		}

		// Draw the prefix and the text.
		if node.textX < width && posY < y+height {
			// Prefix.
			var prefixWidth int
			if len(t.prefixes) > 0 {
				_, prefixWidth = Print(screen, t.prefixes[(node.level-t.topLevel)%len(t.prefixes)], x+node.textX, posY, width-node.textX, AlignLeft, node.color)
			}

			// Text.
			if node.textX+prefixWidth < width {
				style := tcell.StyleDefault.Background(t.backgroundColor).Foreground(node.color)
				if node == t.currentNode {
					style = tcell.StyleDefault.Background(node.color).Foreground(t.backgroundColor)
				}
				printWithStyle(screen, node.text, x+node.textX+prefixWidth, posY, 0, width-node.textX-prefixWidth, AlignLeft, style, false)
			}
		}

		// Advance.
		posY++
	}
}

// InputHandler returns the handler for this primitive.
func (t *TreeView) InputHandler() func(event *tcell.EventKey, setFocus func(p Primitive)) {
	return t.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p Primitive)) {
		selectNode := func() {
			node := t.currentNode
			if node != nil {
				if t.selected != nil {
					t.selected(node)
				}
				if node.selected != nil {
					node.selected()
				}
			}
		}

		// Because the tree is flattened into a list only at drawing time, we also
		// postpone the (selection) movement to drawing time.
		switch key := event.Key(); key {
		case tcell.KeyTab, tcell.KeyBacktab, tcell.KeyEscape:
			if t.done != nil {
				t.done(key)
			}
		case tcell.KeyDown, tcell.KeyRight:
			t.movement = treeDown
		case tcell.KeyUp, tcell.KeyLeft:
			t.movement = treeUp
		case tcell.KeyHome:
			t.movement = treeHome
		case tcell.KeyEnd:
			t.movement = treeEnd
		case tcell.KeyPgDn, tcell.KeyCtrlF:
			t.movement = treePageDown
		case tcell.KeyPgUp, tcell.KeyCtrlB:
			t.movement = treePageUp
		case tcell.KeyRune:
			switch event.Rune() {
			case 'g':
				t.movement = treeHome
			case 'G':
				t.movement = treeEnd
			case 'j':
				t.movement = treeDown
			case 'J':
				t.movement = treeChild
			case 'k':
				t.movement = treeUp
			case 'K':
				t.movement = treeParent
			case ' ':
				selectNode()
			}
		case tcell.KeyEnter:
			selectNode()
		}

		t.process()
	})
}

// MouseHandler returns the mouse handler for this primitive.
func (t *TreeView) MouseHandler() func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
	return t.WrapMouseHandler(func(action MouseAction, event *tcell.EventMouse, setFocus func(p Primitive)) (consumed bool, capture Primitive) {
		x, y := event.Position()
		if !t.InRect(x, y) {
			return false, nil
		}

		switch action {
		case MouseLeftClick:
			setFocus(t)
			_, rectY, _, _ := t.GetInnerRect()
			y += t.offsetY - rectY
			if y >= 0 && y < len(t.nodes) {
				node := t.nodes[y]
				if node.selectable {
					previousNode := t.currentNode
					t.currentNode = node
					if previousNode != node && t.changed != nil {
						t.changed(node)
					}
					if t.selected != nil {
						t.selected(node)
					}
					if node.selected != nil {
						node.selected()
					}
				}
			}
			consumed = true
		case MouseScrollUp:
			t.movement = treeScrollUp
			consumed = true
		case MouseScrollDown:
			t.movement = treeScrollDown
			consumed = true
		}

		return
	})
}
