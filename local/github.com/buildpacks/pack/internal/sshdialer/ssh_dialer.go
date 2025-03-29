// NOTE: this code is based on "github.com/containers/podman/v3/pkg/bindings"

package sshdialer

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	urlPkg "net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/docker/pkg/homedir"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SecretCallback func() (string, error)
type HostKeyCallback func(hostPort string, pubKey ssh.PublicKey) error

type Config struct {
	Identity           string
	PassPhrase         string
	PasswordCallback   SecretCallback
	PassPhraseCallback SecretCallback
	HostKeyCallback    HostKeyCallback
}

const defaultSSHPort = "22"

func NewDialContext(url *urlPkg.URL, config Config) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	sshConfig, err := NewSSHClientConfig(url, config)
	if err != nil {
		return nil, err
	}

	port := url.Port()
	if port == "" {
		port = defaultSSHPort
	}
	host := url.Hostname()

	sshClient, err := ssh.Dial("tcp", net.JoinHostPort(host, port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ssh: %w", err)
	}
	defer func() {
		if sshClient != nil {
			sshClient.Close()
		}
	}()

	var dialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	if url.Path == "" {
		dialContext, err = tryGetStdioDialContext(url, sshClient, config.Identity)
		if err != nil {
			return nil, err
		}
		if dialContext != nil {
			return dialContext, nil
		}
	}

	var addr string
	var network string

	if url.Path != "" {
		addr = url.Path
		network = "unix"
	} else {
		network, addr, err = networkAndAddressFromRemoteDockerHost(sshClient)
		if err != nil {
			return nil, err
		}
	}

	d := dialer{sshClient: sshClient, addr: addr, network: network}
	sshClient = nil
	dialContext = d.DialContext

	runtime.SetFinalizer(&d, func(d *dialer) {
		d.Close()
	})

	return dialContext, nil
}

type dialer struct {
	sshClient *ssh.Client
	network   string
	addr      string
}

func (d *dialer) DialContext(ctx context.Context, n, a string) (net.Conn, error) {
	conn, err := d.Dial(d.network, d.addr)
	if err != nil {
		return nil, err
	}
	go func() {
		if ctx != nil {
			<-ctx.Done()
			conn.Close()
		}
	}()
	return conn, nil
}

func (d *dialer) Dial(n, a string) (net.Conn, error) {
	return d.sshClient.Dial(d.network, d.addr)
}

func (d *dialer) Close() error {
	return d.sshClient.Close()
}

func isWindowsMachine(sshClient *ssh.Client) (bool, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	out, err := session.CombinedOutput("systeminfo")
	if err == nil && strings.Contains(string(out), "Windows") {
		return true, nil
	}
	return false, nil
}

func networkAndAddressFromRemoteDockerHost(sshClient *ssh.Client) (network string, addr string, err error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return network, addr, err
	}
	defer session.Close()

	out, err := session.CombinedOutput("set")
	if err != nil {
		return network, addr, err
	}

	remoteDockerHost := "unix:///var/run/docker.sock"
	isWin, err := isWindowsMachine(sshClient)
	if err != nil {
		return network, addr, err
	}

	if isWin {
		remoteDockerHost = "npipe:////./pipe/docker_engine"
	}

	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "DOCKER_HOST=") {
			parts := strings.SplitN(scanner.Text(), "=", 2)
			remoteDockerHost = strings.Trim(parts[1], `"'`)
			break
		}
	}

	remoteDockerHostURL, err := urlPkg.Parse(remoteDockerHost)
	if err != nil {
		return network, addr, err
	}
	switch remoteDockerHostURL.Scheme {
	case "unix":
		addr = remoteDockerHostURL.Path
	case "fd":
		remoteDockerHostURL.Scheme = "tcp" // don't know why it works that way
		fallthrough
	case "tcp":
		addr = remoteDockerHostURL.Host
	default:
		return "", "", errors.New("scheme is not supported")
	}
	network = remoteDockerHostURL.Scheme

	return network, addr, err
}

func tryGetStdioDialContext(url *urlPkg.URL, sshClient *ssh.Client, identity string) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	session.Stdin = nil
	session.Stdout = nil
	session.Stderr = nil
	err = session.Run("docker system dial-stdio")
	if err == nil {
		var opts []string

		if identity != "" {
			opts = append(opts, "-i", identity)
		}

		connHelper, err := connhelper.GetConnectionHelperWithSSHOpts(url.String(), opts)
		if err != nil {
			return nil, err
		}
		if connHelper != nil {
			return connHelper.Dialer, nil
		}
	}
	return nil, nil
}

func NewSSHClientConfig(url *urlPkg.URL, config Config) (*ssh.ClientConfig, error) {
	var (
		authMethods []ssh.AuthMethod
		signers     []ssh.Signer
	)

	if pw, found := url.User.Password(); found {
		authMethods = append(authMethods, ssh.Password(pw))
	}

	// add signer from explicit identity parameter
	if config.Identity != "" {
		signer, err := loadSignerFromFile(config.Identity, []byte(config.Identity), config.PassPhraseCallback)
		if err != nil {
			return nil, fmt.Errorf("failed to parse identity file: %w", err)
		}
		signers = append(signers, signer)
	}

	// pulls signers (keys) from ssh-agent
	signersFromAgent, err := getSignersFromAgent()
	if err != nil {
		return nil, err
	}
	signers = append(signers, signersFromAgent...)

	// if there is no explicit identity file nor keys from ssh-agent then
	// add keys with standard name from ~/.ssh/
	if len(signers) == 0 {
		defaultKeyPaths := getDefaultKeys()
		if len(defaultKeyPaths) == 1 {
			signer, err := loadSignerFromFile(defaultKeyPaths[0], []byte(config.PassPhrase), config.PassPhraseCallback)
			if err != nil {
				return nil, err
			}
			signers = append(signers, signer)
		}
	}

	authMethods = append(authMethods, signersToAuthMethods(signers)...)

	if len(authMethods) == 0 && config.PasswordCallback != nil {
		authMethods = append(authMethods, ssh.PasswordCallback(config.PasswordCallback))
	}

	const sshTimeout = 5
	clientConfig := &ssh.ClientConfig{
		User:            url.User.Username(),
		Auth:            authMethods,
		HostKeyCallback: createHostKeyCallback(config.HostKeyCallback),
		HostKeyAlgorithms: []string{
			ssh.KeyAlgoECDSA256,
			ssh.KeyAlgoECDSA384,
			ssh.KeyAlgoECDSA521,
			ssh.KeyAlgoED25519,
			ssh.KeyAlgoRSASHA512,
			ssh.KeyAlgoRSASHA256,
			ssh.KeyAlgoRSA,
			ssh.KeyAlgoDSA,
		},
		Timeout: sshTimeout * time.Second,
	}

	return clientConfig, nil
}

// returns signers from ssh agent
func getSignersFromAgent() ([]ssh.Signer, error) {
	if sock, found := os.LookupEnv("SSH_AUTH_SOCK"); found && sock != "" {
		var err error
		var agentSigners []ssh.Signer
		var agentConn net.Conn
		agentConn, err = dialSSHAgent(sock)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to ssh-agent's socket: %w", err)
		}
		agentSigners, err = agent.NewClient(agentConn).Signers()
		if err != nil {
			return nil, fmt.Errorf("failed to get signers from ssh-agent: %w", err)
		}
		return agentSigners, nil
	}
	return nil, nil
}

// Default key names.
var knownKeyNames = []string{"id_rsa", "id_dsa", "id_ecdsa", "id_ecdsa_sk", "id_ed25519", "id_ed25519_sk"}

// returns paths to keys with standard name that are in the ~/.ssh/ directory
func getDefaultKeys() []string {
	var defaultKeyPaths []string
	if home, err := os.UserHomeDir(); err == nil {
		for _, keyName := range knownKeyNames {
			p := filepath.Join(home, ".ssh", keyName)

			fi, err := os.Stat(p)
			if err != nil {
				continue
			}
			if fi.Mode().IsRegular() {
				defaultKeyPaths = append(defaultKeyPaths, p)
			}
		}
	}
	return defaultKeyPaths
}

// transforms slice of singers (keys) into slice of authentication methods for ssh client
func signersToAuthMethods(signers []ssh.Signer) []ssh.AuthMethod {
	if len(signers) == 0 {
		return nil
	}

	var authMethods []ssh.AuthMethod
	dedup := make(map[string]ssh.Signer, len(signers))
	// Dedup signers based on fingerprint, ssh-agent keys override explicit identity
	for _, s := range signers {
		fp := ssh.FingerprintSHA256(s.PublicKey())
		dedup[fp] = s
	}

	var uniq []ssh.Signer
	for _, s := range dedup {
		uniq = append(uniq, s)
	}
	authMethods = append(authMethods, ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		return uniq, nil
	}))

	return authMethods
}

// reads key from given path
// if necessary it will decrypt it
func loadSignerFromFile(path string, passphrase []byte, passPhraseCallback SecretCallback) (ssh.Signer, error) {
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		var missingPhraseError *ssh.PassphraseMissingError
		if ok := errors.As(err, &missingPhraseError); !ok {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		if len(passphrase) == 0 && passPhraseCallback != nil {
			b, err := passPhraseCallback()
			if err != nil {
				return nil, err
			}
			passphrase = []byte(b)
		}

		return ssh.ParsePrivateKeyWithPassphrase(key, passphrase)
	}

	return signer, nil
}

func createHostKeyCallback(userCallback HostKeyCallback) ssh.HostKeyCallback {
	return func(hostPort string, remote net.Addr, pubKey ssh.PublicKey) error {
		knownHosts := filepath.Join(homedir.Get(), ".ssh", "known_hosts")

		fileCallback, err := knownhosts.New(knownHosts)
		if err != nil {
			if os.IsNotExist(err) {
				err = errKeyUnknown
			}
		} else {
			err = fileCallback(hostPort, remote, pubKey)
			if err == nil {
				return nil
			}
		}

		if userCallback != nil {
			err = userCallback(hostPort, pubKey)
			if err == nil {
				return nil
			}
		}

		return err
	}
}

var ErrKeyMismatchMsg = "key mismatch"
var ErrKeyUnknownMsg = "key is unknown"

// I would expose those but since ssh pkg doesn't do correct error wrapping it would be entirely futile
var errKeyUnknown = errors.New(ErrKeyUnknownMsg)
