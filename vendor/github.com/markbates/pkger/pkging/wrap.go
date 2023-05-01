package pkging

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/markbates/pkger/here"
)

func Wrap(parent, with Pkger) Pkger {
	return withPkger{
		base:   with,
		parent: parent,
	}
}

type withPkger struct {
	base   Pkger
	parent Pkger
}

func (w withPkger) String() string {
	if w.parent == nil {
		return fmt.Sprintf("%T", w.base)
	}
	return fmt.Sprintf("%T > %T", w.base, w.parent)
}

func (w withPkger) Parse(p string) (here.Path, error) {
	pt, err := w.base.Parse(p)
	if err != nil {
		if w.parent != nil {
			return w.parent.Parse(p)
		}
		return pt, err
	}
	return pt, nil
}

func (w withPkger) Current() (here.Info, error) {
	pt, err := w.base.Current()
	if err != nil {
		if w.parent != nil {
			return w.parent.Current()
		}
		return pt, err
	}
	return pt, nil
}

func (w withPkger) Info(p string) (here.Info, error) {
	pt, err := w.base.Info(p)
	if err != nil {
		if w.parent != nil {
			return w.parent.Info(p)
		}
		return pt, err
	}
	return pt, nil
}

// Create creates the named file with mode 0666 (before umask) - It's actually 0644, truncating it if it already exists. If successful, methods on the returned File can be used for I/O; the associated file descriptor has mode O_RDWR.
func (w withPkger) Create(p string) (File, error) {
	pt, err := w.base.Create(p)
	if err != nil {
		if w.parent != nil {
			return w.parent.Create(p)
		}
		return pt, err
	}
	return pt, nil
}

// MkdirAll creates a directory named path, along with any necessary parents, and returns nil, or else returns an error. The permission bits perm (before umask) are used for all directories that MkdirAll creates. If path is already a directory, MkdirAll does nothing and returns nil.
func (w withPkger) MkdirAll(p string, perm os.FileMode) error {
	err := w.base.MkdirAll(p, perm)
	if err != nil {
		return err
	}
	if w.parent != nil {
		return w.parent.MkdirAll(p, perm)
	}
	return nil
}

// Open opens the named file for reading. If successful, methods on the returned file can be used for reading; the associated file descriptor has mode O_RDONLY.
func (w withPkger) Open(p string) (File, error) {
	pt, err := w.base.Open(p)
	if err != nil {
		if w.parent != nil {
			return w.parent.Open(p)
		}
		return pt, err
	}
	return pt, nil
}

// Stat returns a FileInfo describing the named file.
func (w withPkger) Stat(p string) (os.FileInfo, error) {
	pt, err := w.base.Stat(p)
	if err != nil {
		if w.parent != nil {
			return w.parent.Stat(p)
		}
		return pt, err
	}
	return pt, nil
}

// Walk walks the file tree rooted at root, calling walkFn for each file or directory in the tree, including root. All errors that arise visiting files and directories are filtered by walkFn. The files are walked in lexical order, which makes the output deterministic but means that for very large directories Walk can be inefficient. Walk does not follow symbolic links. - That is from the standard library. I know. Their grammar teachers can not be happy with them right now.
func (w withPkger) Walk(p string, wf filepath.WalkFunc) error {
	err := w.base.Walk(p, wf)
	if err != nil {
		return err
	}
	if w.parent != nil {
		return w.parent.Walk(p, wf)
	}
	return nil
}

// Remove removes the named file or (empty) directory.
func (w withPkger) Remove(p string) error {
	err := w.base.Remove(p)
	if err != nil {
		return err
	}
	if w.parent != nil {
		return w.parent.Remove(p)
	}
	return nil
}

// RemoveAll removes path and any children it contains. It removes everything it can but returns the first error it encounters. If the path does not exist, RemoveAll returns nil (no error).
func (w withPkger) RemoveAll(p string) error {
	err := w.base.RemoveAll(p)
	if err != nil {
		return err
	}
	if w.parent != nil {
		return w.parent.RemoveAll(p)
	}
	return nil
}
