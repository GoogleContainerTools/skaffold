package stdos

import (
	"net/http"
	"os"
	"path"

	"github.com/markbates/pkger/here"
	"github.com/markbates/pkger/pkging"
)

var _ pkging.File = &File{}

type File struct {
	*os.File
	info   *pkging.FileInfo
	her    here.Info
	path   here.Path
	pkging pkging.Pkger
}

// Close closes the File, rendering it unusable for I/O.
func (f *File) Close() error {
	return f.File.Close()
}

// Info returns the here.Info of the file
func (f *File) Info() here.Info {
	return f.her
}

// Name retuns the name of the file in pkger format
func (f File) Name() string {
	return f.path.String()
}

// Readdir reads the contents of the directory associated with file and returns a slice of up to n FileInfo values, as would be returned by Lstat, in directory order. Subsequent calls on the same file will yield further FileInfos.
//
// If n > 0, Readdir returns at most n FileInfo structures. In this case, if Readdir returns an empty slice, it will return a non-nil error explaining why. At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdir returns all the FileInfo from the directory in a single slice. In this case, if Readdir succeeds (reads all the way to the end of the directory), it returns the slice and a nil error. If it encounters an error before the end of the directory, Readdir returns the FileInfo read until that point and a non-nil error.
func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	return f.File.Readdir(count)
}

// Open implements the http.FileSystem interface. A FileSystem implements access to a collection of named files. The elements in a file path are separated by slash ('/', U+002F) characters, regardless of host operating system convention.
func (f *File) Open(name string) (http.File, error) {
	fp := path.Join(f.Path().Name, name)
	f2, err := f.pkging.Open(fp)
	if err != nil {
		return nil, err
	}
	return f2, nil
}

// Path returns the here.Path of the file
func (f *File) Path() here.Path {
	return f.path
}

// Stat returns the FileInfo structure describing file. If there is an error, it will be of type *PathError.
func (f *File) Stat() (os.FileInfo, error) {
	if f.info != nil {
		return f.info, nil
	}

	info, err := f.File.Stat()
	if err != nil {
		return nil, err
	}
	f.info = pkging.NewFileInfo(info)
	return f.info, nil
}
