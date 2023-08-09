package pkging

import (
	"net/http"
	"os"

	"github.com/markbates/pkger/here"
)

type File interface {
	// Close closes the File, rendering it unusable for I/O.
	Close() error

	// Info returns the here.Info of the file
	Info() here.Info

	// Name retuns the name of the file
	Name() string

	// Open implements the http.FileSystem interface. A FileSystem implements access to a collection of named files. The elements in a file path are separated by slash ('/', U+002F) characters, regardless of host operating system convention.
	Open(name string) (http.File, error)

	// Path returns the here.Path of the file
	Path() here.Path

	// Read reads up to len(b) bytes from the File. It returns the number of bytes read and any error encountered. At end of file, Read returns 0, io.EOF.
	Read(p []byte) (int, error)

	// Readdir reads the contents of the directory associated with file and returns a slice of up to n FileInfo values, as would be returned by Lstat, in directory order. Subsequent calls on the same file will yield further FileInfos.
	//
	// If n > 0, Readdir returns at most n FileInfo structures. In this case, if Readdir returns an empty slice, it will return a non-nil error explaining why. At the end of a directory, the error is io.EOF.
	//
	// If n <= 0, Readdir returns all the FileInfo from the directory in a single slice. In this case, if Readdir succeeds (reads all the way to the end of the directory), it returns the slice and a nil error. If it encounters an error before the end of the directory, Readdir returns the FileInfo read until that point and a non-nil error.
	Readdir(count int) ([]os.FileInfo, error)

	// Seek sets the offset for the next Read or Write on file to offset, interpreted according to whence: 0 means relative to the origin of the file, 1 means relative to the current offset, and 2 means relative to the end. It returns the new offset and an error, if any.
	Seek(offset int64, whence int) (int64, error)

	// Stat returns the FileInfo structure describing file. If there is an error, it will be of type *PathError.
	Stat() (os.FileInfo, error)

	// Write writes len(b) bytes to the File. It returns the number of bytes written and an error, if any. Write returns a non-nil error when n != len(b).
	Write(b []byte) (int, error)
}
