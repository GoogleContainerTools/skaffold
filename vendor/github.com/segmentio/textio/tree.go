package textio

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

// TreeWriter is an implementation of an io.Writer which prints a tree-like
// representation of the content.
//
// Instances of TreeWriter are not safe to use concurrently from multiple
// goroutines.
type TreeWriter struct {
	writer   io.Writer
	children []*TreeWriter
	content  []byte
}

// NewTreeWriter constructs a new TreeWriter which outputs to w. If w is an
// instance of TreeWriter itself the new writer is added to the list of child
// nodes that will be renderend.
func NewTreeWriter(w io.Writer) *TreeWriter {
	node := &TreeWriter{
		writer:  w,
		content: make([]byte, 0, 64),
	}

	if parent, _ := node.Parent().(*TreeWriter); parent != nil {
		if parent.children == nil {
			parent.children = make([]*TreeWriter, 0, 8)
		}
		parent.children = append(parent.children, node)
	}

	return node
}

// Root returns the root of w, which is the node on which calling Close will
// cause the tree to be rendered to the underlying writer.
func (w *TreeWriter) Root() io.Writer {
	if p, _ := w.Parent().(*TreeWriter); p != nil {
		return p.Root()
	}
	return w
}

// Parent returns the parent node of w, which its most direct base of type
// *TreeWriter.
func (w *TreeWriter) Parent() io.Writer {
	if p, _ := w.writer.(*TreeWriter); p != nil {
		return p
	}
	return Parent(w.writer)
}

// Base returns the base writer of w.
func (w *TreeWriter) Base() io.Writer {
	return w.writer
}

// Write writes b to w, satisfies the io.Writer interface.
func (w *TreeWriter) Write(b []byte) (int, error) {
	if w.writer == nil {
		return 0, io.ErrClosedPipe
	}
	w.content = append(w.content, b...)
	return len(b), nil
}

// WriteString writes s to w.
func (w *TreeWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

// WriteByte writes b to w.
func (w *TreeWriter) WriteByte(b byte) error {
	w.content = append(w.content, b)
	return nil
}

// WriteRune writes r to w.
func (w *TreeWriter) WriteRune(r rune) (int, error) {
	b := [8]byte{}
	n := utf8.EncodeRune(b[:], r)
	w.content = append(w.content, b[:n]...)
	return n, nil
}

// Width satisfies the fmt.State interface.
func (w *TreeWriter) Width() (int, bool) {
	f, ok := Base(w.Root()).(fmt.State)
	if ok {
		return f.Width()
	}
	return 0, false
}

// Precision satisfies the fmt.State interface.
func (w *TreeWriter) Precision() (int, bool) {
	f, ok := Base(w.Root()).(fmt.State)
	if ok {
		return f.Precision()
	}
	return 0, false
}

// Flag satisfies the fmt.State interface.
func (w *TreeWriter) Flag(c int) bool {
	f, ok := Base(w.Root()).(fmt.State)
	if ok {
		return f.Flag(c)
	}
	return false
}

// Close closes w, causing all buffered content to be flushed to its underlying
// writer, and future write operations to error with io.ErrClosedPipe.
func (w *TreeWriter) Close() (err error) {
	defer func() {
		w.writer = nil
		switch x := recover().(type) {
		case nil:
		case error:
			err = x
		default:
			err = fmt.Errorf("%+v", x)
		}
	}()

	// Technically we could have each child node write its own representation
	// to a buffer, then render a tree from those content buffers. However this
	// would require a lot more copying because each tree level would be written
	// into the buffer of its parent.
	//
	// Instead the approach we take here only requires 1 level of buffering, no
	// matter how complex the tree is. First the data is buffered into each node
	// and when the tree is closed the code walks through each node and write
	// their content and the leading tree symbols to the underlying writer.

	for _, c := range w.children {
		if err = c.Close(); err != nil {
			return err
		}
	}

	switch w.writer.(type) {
	case nil:
		// Already closed
	case *TreeWriter:
		// Sub-node, don't write anything
	default:
		buffer := [10]string{}
		writer := treeWriter{writer: w.writer, symbols: buffer[:0]}
		writer.writeTree(treeCtx{length: 1}, w)
	}

	return
}

var (
	_ io.Writer       = (*TreeWriter)(nil)
	_ io.StringWriter = (*TreeWriter)(nil)
	_ fmt.State       = (*TreeWriter)(nil)
)

type treeCtx struct {
	index       int  // index of the node
	length      int  // number of nodes
	needNewLine bool // whether a new line must be printed
}

func (ctx *treeCtx) last() bool {
	return ctx.index == (ctx.length - 1)
}

type treeWriter struct {
	writer  io.Writer
	symbols []string
}

func (w *treeWriter) push(ctx treeCtx) {
	w.nextLine(ctx)
	w.symbols = append(w.symbols, "")
}

func (w *treeWriter) pop() {
	w.symbols = w.symbols[:w.lastIndex()]
}

func (w *treeWriter) nextNode(ctx treeCtx) {
	if ctx.last() {
		w.set("└── ")
	} else {
		w.set("├── ")
	}
}

func (w *treeWriter) nextLine(ctx treeCtx) {
	if ctx.last() {
		w.set("    ")
	} else {
		w.set("│   ")
	}
}

func (w *treeWriter) lastIndex() int {
	return len(w.symbols) - 1
}

func (w *treeWriter) empty() bool {
	return len(w.symbols) == 0
}

func (w *treeWriter) set(s string) {
	if !w.empty() {
		w.symbols[w.lastIndex()] = s
	}
}

func (w *treeWriter) writeTree(ctx treeCtx, node *TreeWriter) {
	w.writeNode(ctx, node)
	w.push(ctx)

	ctx.length = len(node.children)
	ctx.needNewLine = !bytes.HasSuffix(node.content, []byte("\n"))

	for i, child := range node.children {
		ctx.index = i
		w.writeTree(ctx, child)
	}

	w.pop()
}

func (w *treeWriter) writeNode(ctx treeCtx, node *TreeWriter) {
	if ctx.needNewLine {
		w.writeString("\n")
		w.nextLine(ctx)
	}

	w.nextNode(ctx)
	i := 0

	forEachLine(node.content, func(line []byte) bool {
		if i != 0 {
			w.nextLine(ctx)
		}

		for _, symbol := range w.symbols {
			w.writeString(symbol)
		}

		w.write(line)
		i++
		return true
	})
}

func (w *treeWriter) writeString(s string) {
	if _, err := io.WriteString(w.writer, s); err != nil {
		panic(err)
	}
}

func (w *treeWriter) write(b []byte) {
	if _, err := w.writer.Write(b); err != nil {
		panic(err)
	}
}
