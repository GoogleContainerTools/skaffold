package sshdialer_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/buildpacks/pack/internal/sshdialer"
	th "github.com/buildpacks/pack/testhelpers"
)

type args struct {
	connStr          string
	credentialConfig sshdialer.Config
}
type testParams struct {
	name        string
	args        args
	setUpEnv    setUpEnvFn
	skipOnWin   bool
	CreateError string
	DialError   string
}

func TestCreateDialer(t *testing.T) {
	for _, privateKey := range []string{"id_ed25519", "id_rsa", "id_dsa"} {
		path := filepath.Join("testdata", privateKey)
		fixupPrivateKeyMod(path)
	}

	defer withoutSSHAgent(t)()
	defer withCleanHome(t)()

	connConfig, cleanUp, err := prepareSSHServer(t)
	th.AssertNil(t, err)

	defer cleanUp()
	time.Sleep(time.Second * 1)

	tests := []testParams{
		{
			name: "read password from input",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d/home/testuser/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{PasswordCallback: func() (string, error) {
					return "idkfa", nil
				}},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig)),
		},
		{
			name: "password in url",
			args: args{connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig)),
		},
		{
			name: "server key is not in known_hosts (the file doesn't exists)",
			args: args{connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv:    all(withoutSSHAgent, withCleanHome),
			CreateError: sshdialer.ErrKeyUnknownMsg,
		},
		{
			name: "server key is not in known_hosts (the file exists)",
			args: args{connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv:    all(withoutSSHAgent, withCleanHome, withEmptyKnownHosts),
			CreateError: sshdialer.ErrKeyUnknownMsg,
		},
		{
			name: "server key is not in known_hosts (the filed doesn't exists) - user force trust",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{HostKeyCallback: func(hostPort string, pubKey ssh.PublicKey) error {
					return nil
				}},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome),
		},
		{
			name: "server key is not in known_hosts (the file exists) - user force trust",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{HostKeyCallback: func(hostPort string, pubKey ssh.PublicKey) error {
					return nil
				}},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withEmptyKnownHosts),
		},
		{
			name: "server key does not match the respective key in known_host",
			args: args{connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv:    all(withoutSSHAgent, withCleanHome, withBadKnownHosts(connConfig)),
			CreateError: sshdialer.ErrKeyMismatchMsg,
		},
		{
			name: "key from identity parameter",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d/home/testuser/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{Identity: filepath.Join("testdata", "id_ed25519")},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig)),
		},
		{
			name: "key at standard location with need to read passphrase",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d/home/testuser/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{PassPhraseCallback: func() (string, error) {
					return "idfa", nil
				}},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKey(t, "id_rsa"), withKnowHosts(connConfig)),
		},
		{
			name: "key at standard location with explicitly set passphrase",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d/home/testuser/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{PassPhrase: "idfa"},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKey(t, "id_rsa"), withKnowHosts(connConfig)),
		},
		{
			name: "key at standard location with no passphrase",
			args: args{connStr: fmt.Sprintf("ssh://testuser@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKey(t, "id_ed25519"), withKnowHosts(connConfig)),
		},
		{
			name: "key from ssh-agent",
			args: args{connStr: fmt.Sprintf("ssh://testuser@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv: all(withGoodSSHAgent, withCleanHome, withKnowHosts(connConfig)),
		},
		{
			name: "password in url with IPv6",
			args: args{connStr: fmt.Sprintf("ssh://testuser:idkfa@[%s]:%d/home/testuser/test.sock",
				connConfig.hostIPv6,
				connConfig.portIPv6,
			)},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig)),
		},
		{
			name: "broken known host",
			args: args{connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv:    all(withoutSSHAgent, withCleanHome, withBrokenKnownHosts),
			CreateError: "missing host pattern",
		},
		{
			name: "inaccessible known host",
			args: args{connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv:    all(withoutSSHAgent, withCleanHome, withInaccessibleKnownHosts),
			skipOnWin:   true,
			CreateError: "permission denied",
		},
		{
			name: "failing pass phrase cbk",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{PassPhraseCallback: func() (string, error) {
					return "", errors.New("test_error_msg")
				}},
			},
			setUpEnv:    all(withoutSSHAgent, withCleanHome, withKey(t, "id_rsa"), withKnowHosts(connConfig)),
			CreateError: "test_error_msg",
		},
		{
			name: "with broken key at default location",
			args: args{connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv:    all(withoutSSHAgent, withCleanHome, withKey(t, "id_dsa"), withKnowHosts(connConfig)),
			CreateError: "failed to parse private key",
		},
		{
			name: "with broken key explicit",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{Identity: filepath.Join("testdata", "id_dsa")},
			},
			setUpEnv:    all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig)),
			CreateError: "failed to parse private key",
		},
		{
			name: "with inaccessible key",
			args: args{connStr: fmt.Sprintf("ssh://testuser:idkfa@%s:%d/home/testuser/test.sock",
				connConfig.hostIPv4,
				connConfig.portIPv4,
			)},
			setUpEnv:    all(withoutSSHAgent, withCleanHome, withInaccessibleKey("id_rsa"), withKnowHosts(connConfig)),
			skipOnWin:   true,
			CreateError: "failed to read key file",
		},
		{
			name: "socket doesn't exist in remote",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d/does/not/exist/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{PasswordCallback: func() (string, error) {
					return "idkfa", nil
				}},
			},
			setUpEnv:  all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig)),
			DialError: "failed to dial unix socket in the remote",
		},
		{
			name: "ssh agent non-existent socket",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d/does/not/exist/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
			},
			setUpEnv:    all(withBadSSHAgentSocket, withCleanHome, withKnowHosts(connConfig)),
			CreateError: "failed to connect to ssh-agent's socket",
		},
		{
			name: "bad ssh agent",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d/does/not/exist/test.sock",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
			},
			setUpEnv:    all(withBadSSHAgent, withCleanHome, withKnowHosts(connConfig)),
			CreateError: "failed to get signers from ssh-agent",
		},
		{
			name: "use docker host from remote unix",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{Identity: filepath.Join("testdata", "id_ed25519")},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig),
				withRemoteDockerHost("unix:///home/testuser/test.sock", connConfig)),
		},
		{
			name: "use docker host from remote tcp",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{Identity: filepath.Join("testdata", "id_ed25519")},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig),
				withRemoteDockerHost("tcp://localhost:1234", connConfig)),
		},
		{
			name: "use docker host from remote fd",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{Identity: filepath.Join("testdata", "id_ed25519")},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig),
				withRemoteDockerHost("fd://localhost:1234", connConfig)),
		},
		{
			name: "use docker host from remote npipe",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{Identity: filepath.Join("testdata", "id_ed25519")},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig),
				withRemoteDockerHost("npipe:////./pipe/docker_engine", connConfig)),
			CreateError: "not supported",
		},
		{
			name: "use emulated windows with default docker host",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{Identity: filepath.Join("testdata", "id_ed25519")},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig),
				withEmulatingWindows(connConfig)),
			CreateError: "not supported",
		},
		{
			name: "use emulated windows with tcp docker host",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{Identity: filepath.Join("testdata", "id_ed25519")},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig), withEmulatingWindows(connConfig),
				withRemoteDockerHost("tcp://localhost:1234", connConfig)),
		},
		{
			name: "use docker system dial-stdio",
			args: args{
				connStr: fmt.Sprintf("ssh://testuser@%s:%d",
					connConfig.hostIPv4,
					connConfig.portIPv4,
				),
				credentialConfig: sshdialer.Config{Identity: filepath.Join("testdata", "id_ed25519")},
			},
			setUpEnv: all(withoutSSHAgent, withCleanHome, withKnowHosts(connConfig), withEmulatedDockerSystemDialStdio(connConfig), withFixedUpSSHCLI),
		},
	}

	for _, ttx := range tests {
		spec.Run(t, "sshDialer/"+ttx.name, testCreateDialer(connConfig, ttx), spec.Report(report.Terminal{}))
	}
}

// this test cannot be parallelized as they use process wide environment variable $HOME
func testCreateDialer(connConfig *SSHServer, tt testParams) func(t *testing.T, when spec.G, it spec.S) {
	return func(t *testing.T, when spec.G, it spec.S) {
		it("creates a dialer", func() {
			u, err := url.Parse(tt.args.connStr)
			th.AssertNil(t, err)

			if net.ParseIP(u.Hostname()).To4() == nil && connConfig.hostIPv6 == "" {
				t.Skip("skipping ipv6 test since test environment doesn't support ipv6 connection")
			}

			if tt.skipOnWin && runtime.GOOS == "windows" {
				t.Skip("skipping this test on windows")
			}

			defer tt.setUpEnv(t)()

			dialContext, err := sshdialer.NewDialContext(u, tt.args.credentialConfig)

			if tt.CreateError == "" {
				th.AssertEq(t, err, nil)
			} else {
				// I wish I could use errors.Is(),
				// however foreign code is not wrapping errors thoroughly
				if err != nil {
					th.AssertContains(t, err.Error(), tt.CreateError)
				} else {
					t.Error("expected error but got nil")
				}
			}
			if err != nil {
				return
			}

			transport := http.Transport{DialContext: dialContext}
			httpClient := http.Client{Transport: &transport}
			defer httpClient.CloseIdleConnections()
			resp, err := httpClient.Get("http://docker/")
			if tt.DialError == "" {
				th.AssertEq(t, err, nil)
			} else {
				// I wish I could use errors.Is(),
				// however foreign code is not wrapping errors thoroughly
				if err != nil {
					th.AssertContains(t, err.Error(), tt.CreateError)
				} else {
					t.Error("expected error but got nil")
				}
			}
			if err != nil {
				return
			}
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			th.AssertTrue(t, err == nil)
			if err != nil {
				return
			}
			th.AssertEq(t, string(b), "OK")
		})
	}
}

// function that prepares testing environment and returns clean up function
// this should be used in conjunction with defer: `defer fn()()`
// e.g. sets environment variables or starts mock up services
// it returns clean up procedure that restores old values of environment variables
// or shuts down mock up services
type setUpEnvFn func(t *testing.T) func()

// combines multiple setUp routines into one setUp routine
func all(fns ...setUpEnvFn) setUpEnvFn {
	return func(t *testing.T) func() {
		t.Helper()
		var cleanUps []func()
		for _, fn := range fns {
			cleanUps = append(cleanUps, fn(t))
		}

		return func() {
			for i := len(cleanUps) - 1; i >= 0; i-- {
				cleanUps[i]()
			}
		}
	}
}

func cp(src, dest string) error {
	srcFs, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("the cp() function failed to stat source file: %w", err)
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("the cp() function failed to read source file: %w", err)
	}

	_, err = os.Stat(dest)
	if err == nil {
		return fmt.Errorf("destination file already exists: %w", os.ErrExist)
	}

	return os.WriteFile(dest, data, srcFs.Mode())
}

// puts key from ./testdata/{keyName} to $HOME/.ssh/{keyName}
// those keys are authorized by the testing ssh server
func withKey(t *testing.T, keyName string) setUpEnvFn {
	t.Helper()

	return func(t *testing.T) func() {
		t.Helper()
		var err error

		home, err := os.UserHomeDir()
		th.AssertNil(t, err)

		err = os.MkdirAll(filepath.Join(home, ".ssh"), 0700)
		th.AssertNil(t, err)

		keySrc := filepath.Join("testdata", keyName)
		keyDest := filepath.Join(home, ".ssh", keyName)
		err = cp(keySrc, keyDest)
		th.AssertNil(t, err)

		return func() {
			os.Remove(keyDest)
		}
	}
}

// withInaccessibleKey creates inaccessible key of give type (specified by keyName)
func withInaccessibleKey(keyName string) setUpEnvFn {
	return func(t *testing.T) func() {
		t.Helper()
		var err error

		home, err := os.UserHomeDir()
		th.AssertNil(t, err)

		err = os.MkdirAll(filepath.Join(home, ".ssh"), 0700)
		th.AssertNil(t, err)

		keyDest := filepath.Join(home, ".ssh", keyName)
		_, err = os.OpenFile(keyDest, os.O_CREATE|os.O_WRONLY, 0000)
		th.AssertNil(t, err)

		return func() {
			os.Remove(keyDest)
		}
	}
}

// sets clean temporary $HOME for test
// this prevents interaction with actual user home which may contain .ssh/
func withCleanHome(t *testing.T) func() {
	t.Helper()
	homeName := "HOME"
	if runtime.GOOS == "windows" {
		homeName = "USERPROFILE"
	}
	tmpDir, err := os.MkdirTemp("", "tmpHome")
	th.AssertNil(t, err)

	oldHome, hadHome := os.LookupEnv(homeName)
	os.Setenv(homeName, tmpDir)

	return func() {
		if hadHome {
			os.Setenv(homeName, oldHome)
		} else {
			os.Unsetenv(homeName)
		}
		os.RemoveAll(tmpDir)
	}
}

// withKnowHosts creates $HOME/.ssh/known_hosts with correct entries
func withKnowHosts(connConfig *SSHServer) setUpEnvFn {
	return func(t *testing.T) func() {
		t.Helper()

		knownHosts := filepath.Join(homedir.Get(), ".ssh", "known_hosts")

		err := os.MkdirAll(filepath.Join(homedir.Get(), ".ssh"), 0700)
		th.AssertNil(t, err)

		_, err = os.Stat(knownHosts)
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			t.Fatal("known_hosts already exists")
		}

		f, err := os.OpenFile(knownHosts, os.O_CREATE|os.O_WRONLY, 0600)
		th.AssertNil(t, err)
		defer f.Close()

		// generate known_hosts
		serverKeysDir := filepath.Join("testdata", "etc", "ssh")
		for _, k := range []string{"ecdsa"} {
			keyPath := filepath.Join(serverKeysDir, fmt.Sprintf("ssh_host_%s_key.pub", k))
			key, err := os.ReadFile(keyPath)
			th.AssertNil(t, err)

			fmt.Fprintf(f, "%s %s", connConfig.hostIPv4, string(key))
			fmt.Fprintf(f, "[%s]:%d %s", connConfig.hostIPv4, connConfig.portIPv4, string(key))

			if connConfig.hostIPv6 != "" {
				fmt.Fprintf(f, "%s %s", connConfig.hostIPv6, string(key))
				fmt.Fprintf(f, "[%s]:%d %s", connConfig.hostIPv6, connConfig.portIPv6, string(key))
			}
		}

		return func() {
			os.Remove(knownHosts)
		}
	}
}

// withBadKnownHosts creates $HOME/.ssh/known_hosts with incorrect entries
func withBadKnownHosts(connConfig *SSHServer) setUpEnvFn {
	return func(t *testing.T) func() {
		t.Helper()

		knownHosts := filepath.Join(homedir.Get(), ".ssh", "known_hosts")

		err := os.MkdirAll(filepath.Join(homedir.Get(), ".ssh"), 0700)
		th.AssertNil(t, err)

		_, err = os.Stat(knownHosts)
		if err == nil || !errors.Is(err, os.ErrNotExist) {
			t.Fatal("known_hosts already exists")
		}

		f, err := os.OpenFile(knownHosts, os.O_CREATE|os.O_WRONLY, 0600)
		th.AssertNil(t, err)
		defer f.Close()

		knownHostTemplate := `{{range $host := .}}{{$host}} ssh-dss AAAAB3NzaC1kc3MAAACBAKH4ufS3ABVb780oTgEL1eu+pI1p6YOq/1KJn5s3zm+L3cXXq76r5OM/roGEYrXWUDGRtfVpzYTAKoMWuqcVc0AZ2zOdYkoy1fSjJ3MqDGF53QEO3TXIUt3gUzmLOewwmZWle0RgMa9GHccv7XVVIZB36RR68ZEUswLaTnlVhXQ1AAAAFQCl4t/LnY7kuUI+tL2qT2XmxmiyqwAAAIB72XaO+LfyIiqBOaTkQf+5rvH1i6y6LDO1QD9pzGWUYw3y03AEveHJMjW0EjnYBKJjK39wcZNTieRyU54lhH/HWeWABn9NcQ3duEf1WSO/s7SPsFO2R6quqVSsStkqf2Yfdy4fl24mH41olwtNA6ft5nkVfkqrIa51si4jU8fBVAAAAIB8SSvyYBcyMGLUlQjzQqhhhAHer9x/1YbknVz+y5PHJLLjHjMC4ZRfLgNEojvMKQW46Te9Pwnudcwv19ho4F+kkCOfss7xjyH70gQm6Sj76DxClmnnPoSRq3qEAOMy5Oh+7vyzxm68KHqd/aOmUaiT1LgqgViS9+kNdCoVMGAMOg== mvasek@bellatrix
{{$host}} ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBLTxVVaQ93ReqHNlbjg5/nBRpuRuG6JIgNeJXWT1V4Dl+dMMrnad3uJBfyrNpvn8rv2qnn6gMTZVtTbLdo96pG0= mvasek@bellatrix
{{$host}} ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOKymJNQszrxetVffPZRfZGKWK786r0mNcg/Wah4+2wn mvasek@bellatrix
{{$host}} ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC/1/OCwec2Gyv5goNYYvos4iOA+a0NolOGsZA/93jmSArPY1zZS1UWeJ6dDTmxGoL/e7jm9lM6NJY7a/zM0C/GqCNRGR/aCUHBJTIgGtH+79FDKO/LWY6ClGY7Lw8qNgZpugbBw3N3HqTtyb2lELhFLT0FEb+le4WUbryooLK2zsz6DnqV4JvTYyyHcanS0h68iSXC7XbkZchvL99l5LT0gD1oDteBPKKFdNOwIjpMkk/IrbFM24xoNkaTDXN87EpQPQzYDfsoGymprc5OZZ8kzrtErQR+yfuunHfzzqDHWi7ga5pbgkuxNt10djWgCfBRsy07FTEgV0JirS0TCfwTBbqRzdjf3dgi8AP+WtkW3mcv4a1XYeqoBo2o9TbfyiA9kERs79UBN0mCe3KNX3Ns0PvutsRLaHmdJ49eaKWkJ6GgL37aqSlIwTixz2xY3eoDSkqHoZpx6Q1MdpSIl5gGVzlaobM/PNM1jqVdyUj+xpjHyiXwHQMKc3eJna7s8Jc= mvasek@bellatrix
{{end}}`

		tmpl := template.New(knownHostTemplate)
		tmpl, err = tmpl.Parse(knownHostTemplate)
		th.AssertNil(t, err)

		hosts := make([]string, 0, 4)
		hosts = append(hosts, connConfig.hostIPv4, fmt.Sprintf("[%s]:%d", connConfig.hostIPv4, connConfig.portIPv4))
		if connConfig.hostIPv6 != "" {
			hosts = append(hosts, connConfig.hostIPv6, fmt.Sprintf("[%s]:%d", connConfig.hostIPv6, connConfig.portIPv4))
		}

		err = tmpl.Execute(f, hosts)
		th.AssertNil(t, err)

		return func() {
			os.Remove(knownHosts)
		}
	}
}

// withBrokenKnownHosts creates broken $HOME/.ssh/known_hosts
func withBrokenKnownHosts(t *testing.T) func() {
	t.Helper()

	knownHosts := filepath.Join(homedir.Get(), ".ssh", "known_hosts")

	err := os.MkdirAll(filepath.Join(homedir.Get(), ".ssh"), 0700)
	th.AssertNil(t, err)

	_, err = os.Stat(knownHosts)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatal("known_hosts already exists")
	}

	f, err := os.OpenFile(knownHosts, os.O_CREATE|os.O_WRONLY, 0600)
	th.AssertNil(t, err)
	defer f.Close()

	_, err = f.WriteString("somegarbage\nsome rubish\n stuff\tqwerty")
	th.AssertNil(t, err)

	return func() {
		os.Remove(knownHosts)
	}
}

// withInaccessibleKnownHosts creates inaccessible $HOME/.ssh/known_hosts
func withInaccessibleKnownHosts(t *testing.T) func() {
	t.Helper()

	knownHosts := filepath.Join(homedir.Get(), ".ssh", "known_hosts")

	err := os.MkdirAll(filepath.Join(homedir.Get(), ".ssh"), 0700)
	th.AssertNil(t, err)

	_, err = os.Stat(knownHosts)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatal("known_hosts already exists")
	}

	f, err := os.OpenFile(knownHosts, os.O_CREATE|os.O_WRONLY, 0000)
	th.AssertNil(t, err)
	defer f.Close()

	return func() {
		os.Remove(knownHosts)
	}
}

// withEmptyKnownHosts creates empty $HOME/.ssh/known_hosts
func withEmptyKnownHosts(t *testing.T) func() {
	t.Helper()

	knownHosts := filepath.Join(homedir.Get(), ".ssh", "known_hosts")

	err := os.MkdirAll(filepath.Join(homedir.Get(), ".ssh"), 0700)
	th.AssertNil(t, err)

	_, err = os.Stat(knownHosts)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatal("known_hosts already exists")
	}

	_, err = os.Create(knownHosts)
	th.AssertNil(t, err)

	return func() {
		os.Remove(knownHosts)
	}
}

// withoutSSHAgent unsets the SSH_AUTH_SOCK environment variable so ssh-agent is not used by test
func withoutSSHAgent(t *testing.T) func() {
	t.Helper()
	oldAuthSock, hadAuthSock := os.LookupEnv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")

	return func() {
		if hadAuthSock {
			os.Setenv("SSH_AUTH_SOCK", oldAuthSock)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}
}

// withBadSSHAgentSocket sets the SSH_AUTH_SOCK environment variable to non-existing file
func withBadSSHAgentSocket(t *testing.T) func() {
	t.Helper()
	oldAuthSock, hadAuthSock := os.LookupEnv("SSH_AUTH_SOCK")
	os.Setenv("SSH_AUTH_SOCK", "/does/not/exists.sock")

	return func() {
		if hadAuthSock {
			os.Setenv("SSH_AUTH_SOCK", oldAuthSock)
		} else {
			os.Unsetenv("SSH_AUTH_SOCK")
		}
	}
}

// withGoodSSHAgent starts serving ssh-agent on temporary unix socket.
// It sets the SSH_AUTH_SOCK environment variable to the temporary socket.
// The agent will return correct keys for the testing ssh server.
func withGoodSSHAgent(t *testing.T) func() {
	t.Helper()

	key, err := os.ReadFile(filepath.Join("testdata", "id_ed25519"))
	th.AssertNil(t, err)

	signer, err := ssh.ParsePrivateKey(key)
	th.AssertNil(t, err)

	return withSSHAgent(t, signerAgent{signer})
}

// withBadSSHAgent starts serving ssh-agent on temporary unix socket.
// It sets the SSH_AUTH_SOCK environment variable to the temporary socket.
// The agent will return incorrect keys for the testing ssh server.
func withBadSSHAgent(t *testing.T) func() {
	return withSSHAgent(t, badAgent{})
}

func withSSHAgent(t *testing.T, ag agent.Agent) func() {
	var err error
	t.Helper()

	var tmpDirForSocket string
	var agentSocketPath string
	if runtime.GOOS == "windows" {
		agentSocketPath = `\\.\pipe\openssh-ssh-agent-test`
	} else {
		tmpDirForSocket, err = os.MkdirTemp("", "forAuthSock")
		th.AssertNil(t, err)

		agentSocketPath = filepath.Join(tmpDirForSocket, "agent.sock")
	}

	unixListener, err := listen(agentSocketPath)
	th.AssertNil(t, err)

	os.Setenv("SSH_AUTH_SOCK", agentSocketPath)

	ctx, cancel := context.WithCancel(context.Background())
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	go func() {
		for {
			conn, err := unixListener.Accept()
			if err != nil {
				errChan <- err

				return
			}

			wg.Add(1)
			go func(conn net.Conn) {
				defer wg.Done()
				go func() {
					<-ctx.Done()
					conn.Close()
				}()
				err := agent.ServeAgent(ag, conn)
				if err != nil {
					if !isErrClosed(err) {
						fmt.Fprintf(os.Stderr, "agent.ServeAgent() failed: %v\n", err)
					}
				}
			}(conn)
		}
	}()

	return func() {
		os.Unsetenv("SSH_AUTH_SOCK")

		err := unixListener.Close()
		th.AssertNil(t, err)

		err = <-errChan

		if !isErrClosed(err) {
			t.Fatal(err)
		}
		cancel()
		wg.Wait()
		if tmpDirForSocket != "" {
			os.RemoveAll(tmpDirForSocket)
		}
	}
}

type signerAgent struct {
	impl ssh.Signer
}

func (a signerAgent) List() ([]*agent.Key, error) {
	return []*agent.Key{{
		Format: a.impl.PublicKey().Type(),
		Blob:   a.impl.PublicKey().Marshal(),
	}}, nil
}

func (a signerAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	return a.impl.Sign(nil, data)
}

func (a signerAgent) Add(key agent.AddedKey) error {
	panic("implement me")
}

func (a signerAgent) Remove(key ssh.PublicKey) error {
	panic("implement me")
}

func (a signerAgent) RemoveAll() error {
	panic("implement me")
}

func (a signerAgent) Lock(passphrase []byte) error {
	panic("implement me")
}

func (a signerAgent) Unlock(passphrase []byte) error {
	panic("implement me")
}

func (a signerAgent) Signers() ([]ssh.Signer, error) {
	panic("implement me")
}

var errBadAgent = errors.New("bad agent error")

type badAgent struct{}

func (b badAgent) List() ([]*agent.Key, error) {
	return nil, errBadAgent
}

func (b badAgent) Sign(key ssh.PublicKey, data []byte) (*ssh.Signature, error) {
	return nil, errBadAgent
}

func (b badAgent) Add(key agent.AddedKey) error {
	return errBadAgent
}

func (b badAgent) Remove(key ssh.PublicKey) error {
	return errBadAgent
}

func (b badAgent) RemoveAll() error {
	return errBadAgent
}

func (b badAgent) Lock(passphrase []byte) error {
	return errBadAgent
}

func (b badAgent) Unlock(passphrase []byte) error {
	return errBadAgent
}

func (b badAgent) Signers() ([]ssh.Signer, error) {
	return nil, errBadAgent
}

// openSSH CLI doesn't take the HOME/USERPROFILE environment variable into account.
// It gets user home in different way (e.g. reading /etc/passwd).
// This means tests cannot mock home dir just by setting environment variable.
// withFixedUpSSHCLI works around the problem, it forces usage of known_hosts from HOME/USERPROFILE.
func withFixedUpSSHCLI(t *testing.T) func() {
	t.Helper()

	sshAbsPath, err := exec.LookPath("ssh")
	th.AssertNil(t, err)

	sshScript := `#!/bin/sh
SSH_BIN -o PasswordAuthentication=no -o ConnectTimeout=3 -o UserKnownHostsFile="$HOME/.ssh/known_hosts" $@
`
	if runtime.GOOS == "windows" {
		sshScript = `@echo off
SSH_BIN -o PasswordAuthentication=no -o ConnectTimeout=3 -o UserKnownHostsFile=%USERPROFILE%\.ssh\known_hosts %*
`
	}
	sshScript = strings.ReplaceAll(sshScript, "SSH_BIN", sshAbsPath)

	home, err := os.UserHomeDir()
	th.AssertNil(t, err)

	homeBin := filepath.Join(home, "bin")
	err = os.MkdirAll(homeBin, 0700)
	th.AssertNil(t, err)

	sshScriptName := "ssh"
	if runtime.GOOS == "windows" {
		sshScriptName = "ssh.bat"
	}

	sshScriptFullPath := filepath.Join(homeBin, sshScriptName)
	err = os.WriteFile(sshScriptFullPath, []byte(sshScript), 0700)
	th.AssertNil(t, err)

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", homeBin+string(os.PathListSeparator)+oldPath)
	return func() {
		os.Setenv("PATH", oldPath)
		os.RemoveAll(homeBin)
	}
}

// withEmulatedDockerSystemDialStdio makes `docker system dial-stdio` viable in the testing ssh server.
// It does so by appending definition of shell function named `docker` into .bashrc .
func withEmulatedDockerSystemDialStdio(sshServer *SSHServer) setUpEnvFn {
	return func(t *testing.T) func() {
		t.Helper()

		oldHasDialStdio := sshServer.HasDialStdio()
		sshServer.SetHasDialStdio(true)
		return func() {
			sshServer.SetHasDialStdio(oldHasDialStdio)
		}
	}
}

// withEmulatingWindows makes changes to the testing ssh server such that
// the server appears to be Windows server for simple check done calling the `systeminfo` command
func withEmulatingWindows(sshServer *SSHServer) setUpEnvFn {
	return func(t *testing.T) func() {
		oldIsWindows := sshServer.IsWindows()
		sshServer.SetIsWindows(true)
		return func() {
			sshServer.SetIsWindows(oldIsWindows)
		}
	}
}

// withRemoteDockerHost makes changes to the testing ssh server such that
// the DOCKER_HOST environment is set to host parameter
func withRemoteDockerHost(host string, sshServer *SSHServer) setUpEnvFn {
	return func(t *testing.T) func() {
		oldHost := sshServer.GetDockerHostEnvVar()
		sshServer.SetDockerHostEnvVar(host)
		return func() {
			sshServer.SetDockerHostEnvVar(oldHost)
		}
	}
}
