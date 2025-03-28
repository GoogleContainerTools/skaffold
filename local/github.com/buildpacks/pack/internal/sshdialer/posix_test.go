//go:build !windows

package sshdialer_test

import (
	"errors"
	"net"
	"os"
)

func fixupPrivateKeyMod(path string) {
	err := os.Chmod(path, 0400)
	if err != nil {
		panic(err)
	}
}

func listen(addr string) (net.Listener, error) {
	return net.Listen("unix", addr)
}

func isErrClosed(err error) bool {
	return errors.Is(err, net.ErrClosed)
}
