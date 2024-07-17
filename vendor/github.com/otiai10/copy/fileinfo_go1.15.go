//go:build !go1.16
// +build !go1.16

package copy

import "os"

// This is a cloned definition of os.FileInfo (go1.15) or fs.FileInfo (go1.16~)
// A FileInfo describes a file and is returned by Stat.
type fileInfo interface {
	// Name() string       // base name of the file
	// Size() int64        // length in bytes for regular files; system-dependent for others
	Mode() os.FileMode // file mode bits
	// ModTime() time.Time // modification time
	IsDir() bool      // abbreviation for Mode().IsDir()
	Sys() interface{} // underlying data source (can return nil)
}
