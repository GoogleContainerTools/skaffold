package pkging

import (
	"os"
	"path/filepath"

	"github.com/markbates/pkger/here"
)

type Pkger interface {
	// Parse the string in here.Path format.
	Parse(p string) (here.Path, error)

	// Current returns the here.Info representing the current Pkger implementation.
	Current() (here.Info, error)

	// Info returns the here.Info of the here.Path
	Info(p string) (here.Info, error)

	// Create creates the named file with mode 0666 (before umask) - It's actually 0644, truncating it if it already exists. If successful, methods on the returned File can be used for I/O; the associated file descriptor has mode O_RDWR.
	Create(name string) (File, error)

	// MkdirAll creates a directory named path, along with any necessary parents, and returns nil, or else returns an error. The permission bits perm (before umask) are used for all directories that MkdirAll creates. If path is already a directory, MkdirAll does nothing and returns nil.
	MkdirAll(p string, perm os.FileMode) error

	// Open opens the named file for reading. If successful, methods on the returned file can be used for reading; the associated file descriptor has mode O_RDONLY.
	// @Parser Directive
	Open(name string) (File, error)

	// Stat returns a FileInfo describing the named file.
	// @Parser Directive
	Stat(name string) (os.FileInfo, error)

	// Walk walks the file tree rooted at root, calling walkFn for each file or directory in the tree, including root. All errors that arise visiting files and directories are filtered by walkFn. The files are walked in lexical order, which makes the output deterministic but means that for very large directories Walk can be inefficient. Walk does not follow symbolic links. - That is from the standard library. I know. Their grammar teachers can not be happy with them right now.
	// @Parser Directive
	Walk(p string, wf filepath.WalkFunc) error

	// Remove removes the named file or (empty) directory.
	Remove(name string) error

	// RemoveAll removes path and any children it contains. It removes everything it can but returns the first error it encounters. If the path does not exist, RemoveAll returns nil (no error).
	RemoveAll(path string) error
}
