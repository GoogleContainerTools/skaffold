//go:build windows

package sshdialer_test

import (
	"errors"
	"net"
	"os/user"
	"strings"

	"github.com/Microsoft/go-winio"
	"github.com/hectane/go-acl"
)

func fixupPrivateKeyMod(path string) {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	mode := uint32(0400)
	err = acl.Apply(path,
		true,
		false,
		acl.GrantName(((mode&0700)<<23)|((mode&0200)<<9), usr.Username))

	// See https://github.com/hectane/go-acl/issues/1
	if err != nil && err.Error() != "The operation completed successfully." {
		panic(err)
	}
}

func listen(addr string) (net.Listener, error) {
	if strings.Contains(addr, "\\pipe\\") {
		return winio.ListenPipe(addr, nil)
	}
	return net.Listen("unix", addr)
}

func isErrClosed(err error) bool {
	return errors.Is(err, net.ErrClosed) || errors.Is(err, winio.ErrPipeListenerClosed) || errors.Is(err, winio.ErrFileClosed)
}
