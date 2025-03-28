package cmd

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	dockerClient "github.com/docker/docker/client"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"github.com/buildpacks/pack/internal/sshdialer"
	"github.com/buildpacks/pack/pkg/client"
)

func tryInitSSHDockerClient() (dockerClient.CommonAPIClient, error) {
	dockerHost := os.Getenv("DOCKER_HOST")
	_url, err := url.Parse(dockerHost)
	isSSH := err == nil && _url.Scheme == "ssh"

	if !isSSH {
		return nil, nil
	}

	credentialsConfig := sshdialer.Config{
		Identity:           os.Getenv("DOCKER_HOST_SSH_IDENTITY"),
		PassPhrase:         os.Getenv("DOCKER_HOST_SSH_IDENTITY_PASSPHRASE"),
		PasswordCallback:   newReadSecretCbk("please enter password:"),
		PassPhraseCallback: newReadSecretCbk("please enter passphrase to private key:"),
		HostKeyCallback:    newHostKeyCbk(),
	}
	dialContext, err := sshdialer.NewDialContext(_url, credentialsConfig)
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		// No tls
		// No proxy
		Transport: &http.Transport{
			DialContext: dialContext,
		},
	}

	dockerClientOpts := []dockerClient.Opt{
		dockerClient.WithVersion(client.DockerAPIVersion),
		dockerClient.WithHTTPClient(httpClient),
		dockerClient.WithHost("http://dummy"),
		dockerClient.WithDialContext(dialContext),
	}

	return dockerClient.NewClientWithOpts(dockerClientOpts...)
}

// readSecret prompts for a secret and returns value input by user from stdin
// Unlike terminal.ReadPassword(), $(echo $SECRET | podman...) is supported.
// Additionally, all input after `<secret>/n` is queued to podman command.
//
// NOTE: this code is based on "github.com/containers/podman/v3/pkg/terminal"
func readSecret(prompt string) (pw []byte, err error) {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		fmt.Fprint(os.Stderr, prompt)
		pw, err = term.ReadPassword(fd)
		fmt.Fprintln(os.Stderr)
		return pw, err
	}

	var b [1]byte
	for {
		n, err := os.Stdin.Read(b[:])
		// terminal.readSecret discards any '\r', so we do the same
		if n > 0 && b[0] != '\r' {
			if b[0] == '\n' {
				return pw, nil
			}
			pw = append(pw, b[0])
			// limit size, so that a wrong input won't fill up the memory
			if len(pw) > 1024 {
				err = errors.New("password too long, 1024 byte limit")
			}
		}
		if err != nil {
			// terminal.readSecret accepts EOF-terminated passwords
			// if non-empty, so we do the same
			if err == io.EOF && len(pw) > 0 {
				err = nil
			}
			return pw, err
		}
	}
}

func newReadSecretCbk(prompt string) sshdialer.SecretCallback {
	var secretSet bool
	var secret string
	return func() (string, error) {
		if secretSet {
			return secret, nil
		}

		p, err := readSecret(prompt)
		if err != nil {
			return "", err
		}
		secretSet = true
		secret = string(p)

		return secret, err
	}
}

func newHostKeyCbk() sshdialer.HostKeyCallback {
	var trust []byte
	return func(hostPort string, pubKey ssh.PublicKey) error {
		if bytes.Equal(trust, pubKey.Marshal()) {
			return nil
		}
		msg := `The authenticity of host %s cannot be established.
%s key fingerprint is %s
Are you sure you want to continue connecting (yes/no)? `
		fmt.Fprintf(os.Stderr, msg, hostPort, pubKey.Type(), ssh.FingerprintSHA256(pubKey))
		reader := bufio.NewReader(os.Stdin)
		answer, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		answer = strings.TrimRight(answer, "\r\n")
		answer = strings.ToLower(answer)

		if answer == "yes" || answer == "y" {
			trust = pubKey.Marshal()
			fmt.Fprintf(os.Stderr, "To avoid this in future add following line into your ~/.ssh/known_hosts:\n%s %s %s\n",
				hostPort, pubKey.Type(), base64.StdEncoding.EncodeToString(trust))
			return nil
		}

		return errors.New("key rejected")
	}
}
