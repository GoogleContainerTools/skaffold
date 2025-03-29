package sshdialer_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHServer struct {
	lock           sync.Locker
	dockerServer   http.Server
	dockerListener listener
	dockerHost     string
	hostIPv4       string
	hostIPv6       string
	portIPv4       int
	portIPv6       int
	hasDialStdio   bool
	isWin          bool
}

func (s *SSHServer) SetIsWindows(v bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.isWin = v
}

func (s *SSHServer) IsWindows() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.isWin
}

func (s *SSHServer) SetDockerHostEnvVar(host string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.dockerHost = host
}

func (s *SSHServer) GetDockerHostEnvVar() string {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.dockerHost
}

func (s *SSHServer) HasDialStdio() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.hasDialStdio
}

func (s *SSHServer) SetHasDialStdio(v bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.hasDialStdio = v
}

const dockerUnixSocket = "/home/testuser/test.sock"
const dockerTCPSocket = "localhost:1234"

// We need to set up SSH server against which we will run the tests.
// This will return SSHServer structure representing the state of the testing server.
// It also returns clean up procedure stopSSH used to shut down the server.
func prepareSSHServer(t *testing.T) (sshServer *SSHServer, stopSSH func(), err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err != nil {
			cancel()
		}
	}()
	httpServerErrChan := make(chan error)
	pollingLoopErr := make(chan error)
	pollingLoopIPv6Err := make(chan error)

	handlePing := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Add("Content-Type", "text/plain")
		writer.WriteHeader(200)
		_, _ = writer.Write([]byte("OK"))
	})

	sshServer = &SSHServer{
		dockerServer: http.Server{
			Handler: handlePing,
		},
		dockerListener: listener{conns: make(chan net.Conn), closed: make(chan struct{})},
		lock:           &sync.Mutex{},
	}

	sshTCPListener, err := net.Listen("tcp4", "localhost:0")
	if err != nil {
		return sshServer, stopSSH, err
	}

	hasIPv6 := true
	sshTCP6Listener, err := net.Listen("tcp6", "localhost:0")
	if err != nil {
		hasIPv6 = false
		t.Log(err)
	}

	host, p, err := net.SplitHostPort(sshTCPListener.Addr().String())
	if err != nil {
		return sshServer, stopSSH, err
	}
	port, err := strconv.ParseInt(p, 10, 32)
	if err != nil {
		return sshServer, stopSSH, err
	}
	sshServer.hostIPv4 = host
	sshServer.portIPv4 = int(port)

	if hasIPv6 {
		host, p, err = net.SplitHostPort(sshTCP6Listener.Addr().String())
		if err != nil {
			return sshServer, stopSSH, err
		}
		port, err = strconv.ParseInt(p, 10, 32)
		if err != nil {
			return sshServer, stopSSH, err
		}
		sshServer.hostIPv6 = host
		sshServer.portIPv6 = int(port)
	}

	t.Logf("Listening on %s", sshTCPListener.Addr())
	if hasIPv6 {
		t.Logf("Listening on %s", sshTCP6Listener.Addr())
	}

	go func() {
		httpServerErrChan <- sshServer.dockerServer.Serve(&sshServer.dockerListener)
	}()

	stopSSH = func() {
		var err error
		cancel()

		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		err = sshServer.dockerServer.Shutdown(stopCtx)
		if err != nil {
			t.Error(err)
		}

		err = <-httpServerErrChan
		if err != nil && !strings.Contains(err.Error(), "Server closed") {
			t.Error(err)
		}

		sshTCPListener.Close()
		err = <-pollingLoopErr
		if err != nil && !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}

		if hasIPv6 {
			sshTCP6Listener.Close()
			err = <-pollingLoopIPv6Err
			if err != nil && !errors.Is(err, net.ErrClosed) {
				t.Error(err)
			}
		}
	}

	connChan := make(chan net.Conn)

	go func() {
		for {
			tcpConn, err := sshTCPListener.Accept()
			if err != nil {
				pollingLoopErr <- err
				return
			}
			connChan <- tcpConn
		}
	}()

	if hasIPv6 {
		go func() {
			for {
				tcpConn, err := sshTCP6Listener.Accept()
				if err != nil {
					pollingLoopIPv6Err <- err
					return
				}
				connChan <- tcpConn
			}
		}()
	}

	go func() {
		for {
			conn := <-connChan
			go func(conn net.Conn) {
				err := sshServer.handleConnection(ctx, conn)
				if err != nil {
					fmt.Fprintf(os.Stderr, "err: %v\n", err)
				}
			}(conn)
		}
	}()

	return sshServer, stopSSH, err
}

func setupServerAuth(conf *ssh.ServerConfig) (err error) {
	passwd := map[string]string{
		"testuser": "idkfa",
		"root":     "iddqd",
	}

	authorizedKeysFiles := []string{"id_ed25519.pub", "id_rsa.pub"}
	authorizedKeys := make(map[[16]byte][]byte, len(authorizedKeysFiles))
	for _, key := range authorizedKeysFiles {
		keyFileName := filepath.Join("testdata", key)
		var bs []byte
		bs, err = os.ReadFile(keyFileName)
		if err != nil {
			return err
		}
		var pk ssh.PublicKey
		pk, _, _, _, err = ssh.ParseAuthorizedKey(bs)
		if err != nil {
			return err
		}

		bs = pk.Marshal()
		authorizedKeys[md5.Sum(bs)] = bs
	}

	*conf = ssh.ServerConfig{
		PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
			if p, ok := passwd[conn.User()]; ok && p == string(password) {
				return nil, nil
			}
			return nil, fmt.Errorf("incorrect password %q for user %q", string(password), conn.User())
		},
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			keyBytes := key.Marshal()
			if b, ok := authorizedKeys[md5.Sum(keyBytes)]; ok && bytes.Equal(b, keyBytes) {
				return &ssh.Permissions{}, nil
			}
			return nil, fmt.Errorf("untrusted public key: %q", string(keyBytes))
		},
	}

	hostKeys := []string{"ssh_host_ecdsa_key", "ssh_host_ed25519_key", "ssh_host_rsa_key"}
	serverKeysDir := filepath.Join("testdata", "etc", "ssh")
	for _, key := range hostKeys {
		keyFileName := filepath.Join(serverKeysDir, key)
		var b []byte
		b, err = os.ReadFile(keyFileName)
		if err != nil {
			return err
		}
		var signer ssh.Signer
		signer, err = ssh.ParsePrivateKey(b)
		if err != nil {
			return err
		}
		conf.AddHostKey(signer)
	}

	return nil
}

func (s *SSHServer) handleConnection(ctx context.Context, conn net.Conn) error {
	var config ssh.ServerConfig
	setupServerAuth(&config)
	sshConn, newChannels, reqs, err := ssh.NewServerConn(conn, &config)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		err = sshConn.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
		}
	}()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		ssh.DiscardRequests(reqs)
	}()

	for newChannel := range newChannels {
		wg.Add(1)
		go func(newChannel ssh.NewChannel) {
			defer wg.Done()
			s.handleChannel(newChannel)
		}(newChannel)
	}

	wg.Wait()

	return nil
}

func (s *SSHServer) handleChannel(newChannel ssh.NewChannel) {
	var err error
	switch newChannel.ChannelType() {
	case "session":
		s.handleSession(newChannel)
	case "direct-streamlocal@openssh.com", "direct-tcpip":
		s.handleTunnel(newChannel)
	default:
		err = newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("type of channel %q is not supported", newChannel.ChannelType()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
		}
	}
}

func (s *SSHServer) handleSession(newChannel ssh.NewChannel) {
	ch, reqs, err := newChannel.Accept()
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
		return
	}

	defer ch.Close()
	for req := range reqs {
		if req.Type == "exec" {
			s.handleExec(ch, req)
			break
		}
	}
}

func (s *SSHServer) handleExec(ch ssh.Channel, req *ssh.Request) {
	var err error
	err = req.Reply(true, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
		return
	}
	execData := struct {
		Command string
	}{}
	err = ssh.Unmarshal(req.Payload, &execData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
		return
	}

	sendExitCode := func(ret uint32) {
		msg := []byte{0, 0, 0, 0}
		binary.BigEndian.PutUint32(msg, ret)
		_, err = ch.SendRequest("exit-status", false, msg)
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
		}
	}

	var ret uint32
	switch {
	case execData.Command == "set":
		ret = 0
		dh := s.GetDockerHostEnvVar()
		if dh != "" {
			_, _ = fmt.Fprintf(ch, "DOCKER_HOST=%s\n", dh)
		}
	case execData.Command == "systeminfo" && s.IsWindows():
		_, _ = fmt.Fprintln(ch, "something Windows something")
		ret = 0
	case execData.Command == "docker system dial-stdio" && s.HasDialStdio():
		pr, pw, conn := newPipeConn()

		select {
		case s.dockerListener.conns <- conn:
		case <-s.dockerListener.closed:
			err = ch.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "err: %v\n", err)
			}
		}

		cpDone := make(chan struct{})
		go func() {
			var err error
			_, err = io.Copy(pw, ch)
			if err != nil {
				fmt.Fprintf(os.Stderr, "err: %v\n", err)
			}
			err = pw.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "err: %v\n", err)
			}
			cpDone <- struct{}{}
		}()

		_, err = io.Copy(ch, pr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
		}
		err = pr.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
		}

		<-cpDone

		<-conn.closed
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
		}

		ret = 0
	default:
		_, _ = fmt.Fprintf(ch.Stderr(), "unknown command: %q\n", execData.Command)
		ret = 127
	}
	sendExitCode(ret)
}

func newPipeConn() (*io.PipeReader, *io.PipeWriter, *rwcConn) {
	pr0, pw0 := io.Pipe()
	pr1, pw1 := io.Pipe()
	rwc := pipeReaderWriterCloser{r: pr0, w: pw1}
	return pr1, pw0, newRWCConn(rwc)
}

type pipeReaderWriterCloser struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func (d pipeReaderWriterCloser) Read(p []byte) (n int, err error) {
	return d.r.Read(p)
}

func (d pipeReaderWriterCloser) Write(p []byte) (n int, err error) {
	return d.w.Write(p)
}

func (d pipeReaderWriterCloser) Close() error {
	err := d.r.Close()
	if err != nil {
		return err
	}
	return d.w.Close()
}

func (s *SSHServer) handleTunnel(newChannel ssh.NewChannel) {
	var err error

	switch newChannel.ChannelType() {
	case "direct-streamlocal@openssh.com":
		bs := newChannel.ExtraData()
		unixExtraData := struct {
			SocketPath string
			Reserved0  string
			Reserved1  uint32
		}{}
		err = ssh.Unmarshal(bs, &unixExtraData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
			return
		}
		if unixExtraData.SocketPath != dockerUnixSocket {
			err = newChannel.Reject(ssh.ConnectionFailed, fmt.Sprintf("bad socket: %q", unixExtraData.SocketPath))
			if err != nil {
				fmt.Fprintf(os.Stderr, "err: %v\n", err)
			}
			return
		}
	case "direct-tcpip":
		bs := newChannel.ExtraData()
		tcpExtraData := struct { //nolint:maligned
			HostLocal  string
			PortLocal  uint32
			HostRemote string
			PortRemote uint32
		}{}
		err = ssh.Unmarshal(bs, &tcpExtraData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
			return
		}

		hostPort := fmt.Sprintf("%s:%d", tcpExtraData.HostLocal, tcpExtraData.PortLocal)
		if hostPort != dockerTCPSocket {
			err = newChannel.Reject(ssh.ConnectionFailed, fmt.Sprintf("bad socket: '%s:%d'", tcpExtraData.HostLocal, tcpExtraData.PortLocal))
			if err != nil {
				fmt.Fprintf(os.Stderr, "err: %v\n", err)
			}
			return
		}
	}

	ch, _, err := newChannel.Accept()
	if err != nil {
		fmt.Fprintf(os.Stderr, "err: %v\n", err)
		return
	}
	conn := newRWCConn(ch)
	select {
	case s.dockerListener.conns <- conn:
	case <-s.dockerListener.closed:
		err = ch.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "err: %v\n", err)
		}
		return
	}
	<-conn.closed
}

type listener struct {
	conns  chan net.Conn
	closed chan struct{}
	o      sync.Once
}

func (l *listener) Accept() (net.Conn, error) {
	select {
	case <-l.closed:
		return nil, net.ErrClosed
	case conn := <-l.conns:
		return conn, nil
	}
}

func (l *listener) Close() error {
	l.o.Do(func() {
		close(l.closed)
	})
	return nil
}

func (l *listener) Addr() net.Addr {
	return &net.UnixAddr{Name: dockerUnixSocket, Net: "unix"}
}

func newRWCConn(rwc io.ReadWriteCloser) *rwcConn {
	return &rwcConn{rwc: rwc, closed: make(chan struct{})}
}

type rwcConn struct {
	rwc    io.ReadWriteCloser
	closed chan struct{}
	o      sync.Once
}

func (c *rwcConn) Read(b []byte) (n int, err error) {
	return c.rwc.Read(b)
}

func (c *rwcConn) Write(b []byte) (n int, err error) {
	return c.rwc.Write(b)
}

func (c *rwcConn) Close() error {
	c.o.Do(func() {
		close(c.closed)
	})
	return c.rwc.Close()
}

func (c *rwcConn) LocalAddr() net.Addr {
	return &net.UnixAddr{Name: dockerUnixSocket, Net: "unix"}
}

func (c *rwcConn) RemoteAddr() net.Addr {
	return &net.UnixAddr{Name: "@", Net: "unix"}
}

func (c *rwcConn) SetDeadline(t time.Time) error { return nil }

func (c *rwcConn) SetReadDeadline(t time.Time) error { return nil }

func (c *rwcConn) SetWriteDeadline(t time.Time) error { return nil }
