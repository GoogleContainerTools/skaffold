package archive

import (
	"archive/tar"
	"path/filepath"
	"strings"
)

type TarReader interface {
	Next() (*tar.Header, error)
	Read(b []byte) (int, error)
}

// NormalizingTarReader read from the wrapped TarReader normalizes header before passing them through to the caller
// NormalizingTarReader always normalizes header.Name so that path separators match the runtime OS
// Other modifications can be enabled by invoking options on the NormalizingTarReader
type NormalizingTarReader struct {
	TarReader
	headerOpts    []HeaderOpt
	excludedPaths []string
}

// Strip removes leading directories for any subsequently read *tar.Header
func (tr *NormalizingTarReader) Strip(prefix string) {
	tr.headerOpts = append(tr.headerOpts, func(header *tar.Header) *tar.Header {
		header.Name = strings.TrimPrefix(header.Name, prefix)
		return header
	})
}

// ExcludedPaths configures an array of paths to be skipped on subsequent calls to Next
// Children of the configured paths will also be skipped
// paths should match the unmodified Name of the *tar.Header returned by the wrapped TarReader, not the normalized headers
func (tr *NormalizingTarReader) ExcludePaths(paths []string) {
	tr.excludedPaths = paths
}

// PrependDir will set the Name of any subsequently read *tar.Header the result of filepath.Join of dir and the
//  original Name
func (tr *NormalizingTarReader) PrependDir(dir string) {
	tr.headerOpts = append(tr.headerOpts, func(hdr *tar.Header) *tar.Header {
		// Suppress gosec check for zip slip vulnerability, as we set dir in our code.
		// #nosec G305
		hdr.Name = filepath.Join(dir, hdr.Name)
		return hdr
	})
}

// NewNormalizingTarReader creates a NormalizingTarReaders that wraps the provided TarReader
func NewNormalizingTarReader(tr TarReader) *NormalizingTarReader {
	return &NormalizingTarReader{TarReader: tr}
}

// Next calls Next on the wrapped TarReader and applies modifications before returning the *tar.Header
// If the wrapped TarReader returns a *tar.Header matching an excluded path Next will proceed to the next entry,
//  returning the first non-excluded entry
// Modification options will be apply in the order the options were invoked.
// Standard modifications (path separators normalization) are applied last.
func (tr *NormalizingTarReader) Next() (*tar.Header, error) {
	hdr, err := tr.TarReader.Next()
	if err != nil {
		return nil, err
	}
	for _, excluded := range tr.excludedPaths {
		if strings.HasPrefix(hdr.Name, excluded) {
			return tr.Next() // If path is excluded move on to the next entry
		}
	}
	for _, opt := range tr.headerOpts {
		hdr = opt(hdr)
	}
	if hdr.Name == "" {
		return tr.Next() // If entire path is stripped move on to the next entry
	}
	hdr.Name = filepath.FromSlash(hdr.Name)
	return hdr, nil
}
