//go:build unix

package sshdialer

import "net"

func dialSSHAgent(addr string) (net.Conn, error) {
	return net.Dial("unix", addr)
}
