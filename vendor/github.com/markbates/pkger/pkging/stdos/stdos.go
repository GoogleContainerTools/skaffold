package stdos

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/markbates/pkger/here"
	"github.com/markbates/pkger/internal/maps"
	"github.com/markbates/pkger/pkging"
)

var _ pkging.Pkger = &Pkger{}

type Pkger struct {
	Here  here.Info
	infos *maps.Infos
}

// New returns *Pkger for the provided here.Info
func New(her here.Info) (*Pkger, error) {
	p := &Pkger{
		infos: &maps.Infos{},
		Here:  her,
	}
	p.infos.Store(her.ImportPath, her)
	return p, nil
}

// Create creates the named file with mode 0666 (before umask) - It's actually 0644, truncating it if it already exists. If successful, methods on the returned File can be used for I/O; the associated file descriptor has mode O_RDWR.
func (fx *Pkger) Create(name string) (pkging.File, error) {
	pt, err := fx.Parse(name)
	if err != nil {
		return nil, err
	}

	her, err := fx.Info(pt.Pkg)
	if err != nil {
		return nil, err
	}

	name = filepath.Join(her.Dir, pt.Name)
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}

	nf := &File{
		File:   f,
		her:    her,
		path:   pt,
		pkging: fx,
	}

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	nf.info = pkging.NewFileInfo(info)
	return nf, nil
}

// Current returns the here.Info representing the current Pkger implementation.
func (f *Pkger) Current() (here.Info, error) {
	return f.Here, nil
}

// Info returns the here.Info of the here.Path
func (f *Pkger) Info(p string) (here.Info, error) {
	info, ok := f.infos.Load(p)
	if ok {
		return info, nil
	}

	info, err := here.Package(p)
	if err != nil {
		return info, err
	}
	f.infos.Store(p, info)
	return info, nil
}

// MkdirAll creates a directory named path, along with any necessary parents, and returns nil, or else returns an error. The permission bits perm (before umask) are used for all directories that MkdirAll creates. If path is already a directory, MkdirAll does nothing and returns nil.
func (f *Pkger) MkdirAll(p string, perm os.FileMode) error {
	pt, err := f.Parse(p)
	if err != nil {
		return err
	}
	info, err := f.Info(pt.Pkg)
	if err != nil {
		return err
	}
	return os.MkdirAll(filepath.Join(info.Dir, p), perm)
}

// Open opens the named file for reading. If successful, methods on the returned file can be used for reading; the associated file descriptor has mode O_RDONLY.
func (fx *Pkger) Open(name string) (pkging.File, error) {
	pt, err := fx.Parse(name)
	if err != nil {
		return nil, err
	}

	her, err := fx.Info(pt.Pkg)
	if err != nil {
		return nil, err
	}

	name = filepath.Join(her.Dir, pt.Name)
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	nf := &File{
		File:   f,
		her:    her,
		path:   pt,
		pkging: fx,
	}

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	nf.info = pkging.NewFileInfo(info)

	return nf, nil
}

// Parse the string in here.Path format.
func (f *Pkger) Parse(p string) (here.Path, error) {
	return f.Here.Parse(p)
}

// Stat returns a FileInfo describing the named file.
func (fx *Pkger) Stat(name string) (os.FileInfo, error) {
	pt, err := fx.Parse(name)
	if err != nil {
		return nil, err
	}

	her, err := fx.Info(pt.Pkg)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(filepath.Join(her.Dir, pt.Name))
	if err != nil {
		return nil, err
	}

	info = pkging.NewFileInfo(info)

	return info, nil
}

// Walk walks the file tree rooted at root, calling walkFn for each file or directory in the tree, including root. All errors that arise visiting files and directories are filtered by walkFn. The files are walked in lexical order, which makes the output deterministic but means that for very large directories Walk can be inefficient. Walk does not follow symbolic links. - That is from the standard library. I know. Their grammar teachers can not be happy with them right now.
func (f *Pkger) Walk(p string, wf filepath.WalkFunc) error {
	pt, err := f.Parse(p)
	if err != nil {
		return err
	}

	info, err := f.Info(pt.Pkg)
	if err != nil {
		return err
	}

	fp := filepath.Join(info.Dir, pt.Name)
	err = filepath.Walk(fp, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		pt, err := f.Parse(fmt.Sprintf("%s:%s", pt.Pkg, path))
		if err != nil {
			return err
		}

		info, err := f.Info(pt.Pkg)
		if err != nil {
			return err
		}

		path = strings.TrimPrefix(path, info.Dir)
		path = strings.ReplaceAll(path, "\\", "/")
		pt.Name = path
		return wf(pt.String(), pkging.NewFileInfo(fi), nil)
	})

	return err
}

// Remove removes the named file or (empty) directory.
func (fx *Pkger) Remove(name string) error {
	pt, err := fx.Parse(name)
	if err != nil {
		return err
	}

	info, err := fx.Info(pt.Pkg)
	if err != nil {
		return err
	}

	return os.Remove(filepath.Join(info.Dir, pt.Name))
}

// RemoveAll removes path and any children it contains. It removes everything it can but returns the first error it encounters. If the path does not exist, RemoveAll returns nil (no error).
func (fx *Pkger) RemoveAll(name string) error {
	pt, err := fx.Parse(name)
	if err != nil {
		return err
	}

	info, err := fx.Info(pt.Pkg)
	if err != nil {
		return err
	}

	return os.RemoveAll(filepath.Join(info.Dir, pt.Name))
}
