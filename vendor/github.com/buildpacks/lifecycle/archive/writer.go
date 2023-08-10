package archive

import (
	"archive/tar"
	"path/filepath"
	"strings"
	"time"
)

var (
	// NormalizedModTime provides a valid "zero" value for ModTime
	NormalizedModTime = time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)
)

type TarWriter interface {
	WriteHeader(hdr *tar.Header) error
	Write(b []byte) (int, error)
	Close() error
}

// NormalizingTarWriter normalizes any written *tar.Header before passing it through to the wrapped TarWriter
// NormalizingTarWriter always normalizes ModTime, Uname, and Gname
// Other modifications can be enabled by invoking options on the NormalizingTarWriter
type NormalizingTarWriter struct {
	TarWriter
	headerOpts []HeaderOpt
}

type HeaderOpt func(header *tar.Header) *tar.Header

// WithUID sets Uid of any subsequently written *tar.Header to uid
func (tw *NormalizingTarWriter) WithUID(uid int) {
	tw.headerOpts = append(tw.headerOpts, func(hdr *tar.Header) *tar.Header {
		hdr.Uid = uid
		return hdr
	})
}

// WithGID sets Gid of any subsequently written *tar.Header to gid
func (tw *NormalizingTarWriter) WithGID(gid int) {
	tw.headerOpts = append(tw.headerOpts, func(hdr *tar.Header) *tar.Header {
		hdr.Gid = gid
		return hdr
	})
}

// WithModTime sets the ModTime of any subsequently written *tar.Header to modTime
func (tw *NormalizingTarWriter) WithModTime(modTime time.Time) {
	tw.headerOpts = append(tw.headerOpts, func(hdr *tar.Header) *tar.Header {
		hdr.ModTime = modTime
		return hdr
	})
}

// NewNormalizingTarWriter creates a NormalizingTarWriter that wraps the provided TarWriter
func NewNormalizingTarWriter(tw TarWriter) *NormalizingTarWriter {
	return &NormalizingTarWriter{tw, []HeaderOpt{}}
}

// WriteHeader writes the header to the wrapped TarWriter after applying standard and configured modifications
// Modification options will be apply in the order the options were invoked.
// Standard modification (ModTime, Uname, and Gname) are applied last.
func (tw *NormalizingTarWriter) WriteHeader(hdr *tar.Header) error {
	for _, opt := range tw.headerOpts {
		hdr = opt(hdr)
	}
	hdr.Name = filepath.ToSlash(strings.TrimPrefix(hdr.Name, filepath.VolumeName(hdr.Name)))
	hdr.Uname = ""
	hdr.Gname = ""
	return tw.TarWriter.WriteHeader(hdr)
}
