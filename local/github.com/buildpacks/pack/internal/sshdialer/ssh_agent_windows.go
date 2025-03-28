package sshdialer

import (
	"net"
	"strings"

	"github.com/Microsoft/go-winio"
)

func dialSSHAgent(addr string) (net.Conn, error) {
	if strings.Contains(addr, "\\pipe\\") {
		return winio.DialPipe(addr, nil)
	}
	return net.Dial("unix", addr)
}
