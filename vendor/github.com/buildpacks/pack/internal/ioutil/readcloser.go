package ioutil

import "io"

type readCloserWrapper struct {
	io.Reader
	closeFn func() error
}

// NewReadCloserWrapper returns an io.ReadCloser that reads from r and runs
// closeFn on Close. It replaces github.com/docker/docker/pkg/ioutils.NewReadCloserWrapper,
// which is not part of the moby/moby split modules.
func NewReadCloserWrapper(r io.Reader, closeFn func() error) io.ReadCloser {
	return &readCloserWrapper{Reader: r, closeFn: closeFn}
}

func (w *readCloserWrapper) Close() error {
	return w.closeFn()
}
