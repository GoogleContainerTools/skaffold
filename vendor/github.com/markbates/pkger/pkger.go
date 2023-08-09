package pkger

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/markbates/pkger/here"
	"github.com/markbates/pkger/pkging"
	"github.com/markbates/pkger/pkging/stdos"
)

var current pkging.Pkger
var gil = &sync.RWMutex{}

var disk = func() pkging.Pkger {
	her, _ := here.Current()
	n, _ := stdos.New(her)
	return n
}()

func impl() pkging.Pkger {
	gil.RLock()
	defer gil.RUnlock()
	if current == nil {
		return disk
	}
	return current
}

type Dir string

func (d Dir) Open(name string) (http.File, error) {
	f, err := impl().Open(string(d))
	if err != nil {
		return nil, err
	}
	return f.Open(name)
}

// Parse the string in here.Path format.
func Parse(p string) (here.Path, error) {
	return impl().Parse(p)
}

// Current returns the here.Info representing the current Pkger implementation.
func Current() (here.Info, error) {
	return impl().Current()
}

// Info returns the here.Info of the here.Path
func Info(p string) (here.Info, error) {
	return impl().Info(p)
}

// Create creates the named file with mode 0666 (before umask) - It's actually 0644, truncating it if it already exists. If successful, methods on the returned File can be used for I/O; the associated file descriptor has mode O_RDWR.
func Create(p string) (pkging.File, error) {
	return impl().Create(p)
}

// MkdirAll creates a directory named path, along with any necessary parents, and returns nil, or else returns an error. The permission bits perm (before umask) are used for all directories that MkdirAll creates. If path is already a directory, MkdirAll does nothing and returns nil.
func MkdirAll(p string, perm os.FileMode) error {
	return impl().MkdirAll(p, perm)
}

// Open opens the named file for reading. If successful, methods on the returned file can be used for reading; the associated file descriptor has mode O_RDONLY.
func Open(p string) (pkging.File, error) {
	return impl().Open(p)
}

// Stat returns a FileInfo describing the named file.
func Stat(name string) (os.FileInfo, error) {
	return impl().Stat(name)
}

// Walk walks the file tree rooted at root, calling walkFn for each file or directory in the tree, including root. All errors that arise visiting files and directories are filtered by walkFn. The files are walked in lexical order, which makes the output deterministic but means that for very large directories Walk can be inefficient. Walk does not follow symbolic links. - That is from the standard library. I know. Their grammar teachers can not be happy with them right now.
func Walk(p string, wf filepath.WalkFunc) error {
	return impl().Walk(p, wf)
}

// Remove removes the named file or (empty) directory.
func Remove(name string) error {
	return impl().Remove(name)
}

// RemoveAll removes path and any children it contains. It removes everything it can but returns the first error it encounters. If the path does not exist, RemoveAll returns nil (no error).
func RemoveAll(name string) error {
	return impl().RemoveAll(name)
}

// Include is a no-op that directs the pkger tool to include the desired file or folder.
func Include(name string) string {
	return name
}
