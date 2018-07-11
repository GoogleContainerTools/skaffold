// A `pass` based credential helper. Passwords are stored as arguments to pass
// of the form: "$PASS_FOLDER/base64-url(serverURL)/username". We base64-url
// encode the serverURL, because under the hood pass uses files and folders, so
// /s will get translated into additional folders.
package pass

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"

	"github.com/docker/docker-credential-helpers/credentials"
)

const PASS_FOLDER = "docker-credential-helpers"

// Pass handles secrets using Linux secret-service as a store.
type Pass struct{}

// Ideally these would be stored as members of Pass, but since all of Pass's
// methods have value receivers, not pointer receivers, and changing that is
// backwards incompatible, we assume that all Pass instances share the same configuration

// initializationMutex is held while initializing so that only one 'pass'
// round-tripping is done to check pass is functioning.
var initializationMutex sync.Mutex
var passInitialized bool

// CheckInitialized checks whether the password helper can be used. It
// internally caches and so may be safely called multiple times with no impact
// on performance, though the first call may take longer.
func (p Pass) CheckInitialized() bool {
	return p.checkInitialized() == nil
}

func (p Pass) checkInitialized() error {
	initializationMutex.Lock()
	defer initializationMutex.Unlock()
	if passInitialized {
		return nil
	}
	// In principle, we could just run `pass init`. However, pass has a bug
	// where if gpg fails, it doesn't always exit 1. Additionally, pass
	// uses gpg2, but gpg is the default, which may be confusing. So let's
	// just explictily check that pass actually can store and retreive a
	// password.
	password := "pass is initialized"
	name := path.Join(getPassDir(), "docker-pass-initialized-check")

	_, err := p.runPassHelper(password, "insert", "-f", "-m", name)
	if err != nil {
		return fmt.Errorf("error initializing pass: %v", err)
	}

	stored, err := p.runPassHelper("", "show", name)
	if err != nil {
		return fmt.Errorf("error fetching password during initialization: %v", err)
	}
	if stored != password {
		return fmt.Errorf("error round-tripping password during initialization: %q != %q", password, stored)
	}
	passInitialized = true
	return nil
}

func (p Pass) runPass(stdinContent string, args ...string) (string, error) {
	if err := p.checkInitialized(); err != nil {
		return "", err
	}
	return p.runPassHelper(stdinContent, args...)
}

func (p Pass) runPassHelper(stdinContent string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("pass", args...)
	cmd.Stdin = strings.NewReader(stdinContent)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, stderr.String())
	}

	// trim newlines; pass v1.7.1+ includes a newline at the end of `show` output
	return strings.TrimRight(stdout.String(), "\n\r"), nil
}

// Add adds new credentials to the keychain.
func (h Pass) Add(creds *credentials.Credentials) error {
	if creds == nil {
		return errors.New("missing credentials")
	}

	encoded := base64.URLEncoding.EncodeToString([]byte(creds.ServerURL))

	_, err := h.runPass(creds.Secret, "insert", "-f", "-m", path.Join(PASS_FOLDER, encoded, creds.Username))
	return err
}

// Delete removes credentials from the store.
func (h Pass) Delete(serverURL string) error {
	if serverURL == "" {
		return errors.New("missing server url")
	}

	encoded := base64.URLEncoding.EncodeToString([]byte(serverURL))
	_, err := h.runPass("", "rm", "-rf", path.Join(PASS_FOLDER, encoded))
	return err
}

func getPassDir() string {
	passDir := "$HOME/.password-store"
	if envDir := os.Getenv("PASSWORD_STORE_DIR"); envDir != "" {
		passDir = envDir
	}
	return os.ExpandEnv(passDir)
}

// listPassDir lists all the contents of a directory in the password store.
// Pass uses fancy unicode to emit stuff to stdout, so rather than try
// and parse this, let's just look at the directory structure instead.
func listPassDir(args ...string) ([]os.FileInfo, error) {
	passDir := getPassDir()
	p := path.Join(append([]string{passDir, PASS_FOLDER}, args...)...)
	contents, err := ioutil.ReadDir(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []os.FileInfo{}, nil
		}

		return nil, err
	}

	return contents, nil
}

// Get returns the username and secret to use for a given registry server URL.
func (h Pass) Get(serverURL string) (string, string, error) {
	if serverURL == "" {
		return "", "", errors.New("missing server url")
	}

	encoded := base64.URLEncoding.EncodeToString([]byte(serverURL))

	if _, err := os.Stat(path.Join(getPassDir(), PASS_FOLDER, encoded)); err != nil {
		if os.IsNotExist(err) {
			return "", "", nil
		}

		return "", "", err
	}

	usernames, err := listPassDir(encoded)
	if err != nil {
		return "", "", err
	}

	if len(usernames) < 1 {
		return "", "", fmt.Errorf("no usernames for %s", serverURL)
	}

	actual := strings.TrimSuffix(usernames[0].Name(), ".gpg")
	secret, err := h.runPass("", "show", path.Join(PASS_FOLDER, encoded, actual))
	return actual, secret, err
}

// List returns the stored URLs and corresponding usernames for a given credentials label
func (h Pass) List() (map[string]string, error) {
	servers, err := listPassDir()
	if err != nil {
		return nil, err
	}

	resp := map[string]string{}

	for _, server := range servers {
		if !server.IsDir() {
			continue
		}

		serverURL, err := base64.URLEncoding.DecodeString(server.Name())
		if err != nil {
			return nil, err
		}

		usernames, err := listPassDir(server.Name())
		if err != nil {
			return nil, err
		}

		if len(usernames) < 1 {
			return nil, fmt.Errorf("no usernames for %s", serverURL)
		}

		resp[string(serverURL)] = strings.TrimSuffix(usernames[0].Name(), ".gpg")
	}

	return resp, nil
}
