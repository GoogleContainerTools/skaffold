# textio [![CircleCI](https://circleci.com/gh/segmentio/textio.svg?style=shield)](https://circleci.com/gh/segmentio/textio) [![Go Report Card](https://goreportcard.com/badge/github.com/segmentio/textio)](https://goreportcard.com/report/github.com/segmentio/textio) [![GoDoc](https://godoc.org/github.com/segmentio/textio?status.svg)](https://godoc.org/github.com/segmentio/textio)
Go package providing tools for advanced text manipulations

## Motivation

This package aims to provide a sutie of tools to deal with text parsing and
formatting. It is intended to extend what the standard library already offers,
and make it easy to integrate with it.

## Examples

This sections presents a couple of examples about how to use this package.

### Indenting

Indentation is often a complex problem to solve when dealing with stream of text
that may be composed of multiple lines. To address this problem, this package
provides the `textio.PrefixWriter` type, which implements the `io.Writer`
interface and automatically prepends every line of output with a predefined
prefix.

Here is an example:
```go
func copyIndent(w io.Writer, r io.Reader) error {
    p := textio.NewPrefixWriter(w, "\t")

    // Copy data from an input stream into the PrefixWriter, all lines will
    // be prefixed with a '\t' character.
    if _, err := io.Copy(p, r); err != nil {
        return err
    }

    // Flushes any data buffered in the PrefixWriter, this is important in
    // case the last line was not terminated by a '\n' character.
    return p.Flush()
}
```

### Tree Formatting

A common way to represent tree-like structures is the formatting used by the
`tree(1)` unix command. The `textio.TreeWriter` type is an implementation of
an `io.Writer` which supports this kind of output. It works in a recursive
fashion where nodes created from a parent tree writer are formatted as part
of that tree structure.

Here is an example:
```go
func ls(w io.Writer, path string) {
	tree := NewTreeWriter(w)
	tree.WriteString(filepath.Base(path))
	defer tree.Close()

	files, _ := ioutil.ReadDir(path)

	for _, f := range files {
		if f.Mode().IsDir() {
			ls(tree, filepath.Join(path, f.Name()))
		}
	}

	for _, f := range files {
		if !f.Mode().IsDir() {
			io.WriteString(NewTreeWriter(tree), f.Name())
		}
	}
}

...

ls(os.Stdout, "examples")
```
Which gives this output:
```
examples
├── A
│   ├── 1
│   └── 2
└── message
```
