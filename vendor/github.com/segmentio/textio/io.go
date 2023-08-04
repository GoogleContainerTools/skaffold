package textio

import "io"

// Base returns the direct base of w, which may be w itself if it had no base
// writer.
func Base(w io.Writer) io.Writer {
	if d, ok := w.(decorator); ok {
		return coalesceWriters(d.Base(), w)
	}
	return w
}

// Root returns the root writer of w, which is found by going up the list of
// base writers.
//
// The node is usually the writer where the content ends up being written.
func Root(w io.Writer) io.Writer {
	switch x := w.(type) {
	case tree:
		return coalesceWriters(x.Root(), w)
	case node:
		return coalesceWriters(Root(x.Parent()), w)
	case decorator:
		return coalesceWriters(Root(x.Base()), w)
	default:
		return w
	}
}

// Parent returns the parent writer of w, which is usually a writer of a similar
// type on tree-like writer structures.
func Parent(w io.Writer) io.Writer {
	switch x := w.(type) {
	case node:
		return coalesceWriters(x.Parent(), w)
	case decorator:
		return coalesceWriters(Parent(x.Base()), w)
	default:
		return x
	}
}

type decorator interface {
	Base() io.Writer
}

type node interface {
	Parent() io.Writer
}

type tree interface {
	Root() io.Writer
}

func coalesceWriters(writers ...io.Writer) io.Writer {
	for _, w := range writers {
		if w != nil {
			return w
		}
	}
	return nil
}
