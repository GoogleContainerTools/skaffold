package client

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/docker/docker-credential-helpers/credentials"
)

const (
	validServerAddress   = "https://index.docker.io/v1"
	validUsername        = "linus"
	validServerAddress2  = "https://example.com:5002"
	invalidServerAddress = "https://foobar.example.com"
	missingCredsAddress  = "https://missing.docker.io/v1"
)

var errProgramExited = fmt.Errorf("exited 1")

// mockProgram simulates interactions between the docker client and a remote
// credentials helper.
// Unit tests inject this mocked command into the remote to control execution.
type mockProgram struct {
	arg   string
	input io.Reader
}

// Output returns responses from the remote credentials helper.
// It mocks those responses based in the input in the mock.
func (m *mockProgram) Output() ([]byte, error) {
	in, err := ioutil.ReadAll(m.input)
	if err != nil {
		return nil, err
	}
	inS := string(in)

	switch m.arg {
	case "erase":
		switch inS {
		case validServerAddress:
			return nil, nil
		default:
			return []byte("program failed"), errProgramExited
		}
	case "get":
		switch inS {
		case validServerAddress:
			return []byte(`{"Username": "foo", "Secret": "bar"}`), nil
		case validServerAddress2:
			return []byte(`{"Username": "<token>", "Secret": "abcd1234"}`), nil
		case missingCredsAddress:
			return []byte(credentials.NewErrCredentialsNotFound().Error()), errProgramExited
		case invalidServerAddress:
			return []byte("program failed"), errProgramExited
		case "":
			return []byte(credentials.NewErrCredentialsMissingServerURL().Error()), errProgramExited
		}
	case "store":
		var c credentials.Credentials
		err := json.NewDecoder(strings.NewReader(inS)).Decode(&c)
		if err != nil {
			return []byte("error storing credentials"), errProgramExited
		}
		switch c.ServerURL {
		case validServerAddress:
			return nil, nil
		case validServerAddress2:
			return nil, nil
		default:
			return []byte("error storing credentials"), errProgramExited
		}
	case "list":
		return []byte(fmt.Sprintf(`{"%s": "%s"}`, validServerAddress, validUsername)), nil

	}

	return []byte(fmt.Sprintf("unknown argument %q with %q", m.arg, inS)), errProgramExited
}

// Input sets the input to send to a remote credentials helper.
func (m *mockProgram) Input(in io.Reader) {
	m.input = in
}

func mockProgramFn(args ...string) Program {
	return &mockProgram{
		arg: args[0],
	}
}

func ExampleStore() {
	p := NewShellProgramFunc("docker-credential-secretservice")

	c := &credentials.Credentials{
		ServerURL: "https://example.com",
		Username:  "calavera",
		Secret:    "my super secret token",
	}

	if err := Store(p, c); err != nil {
		fmt.Println(err)
	}
}

func TestStore(t *testing.T) {
	valid := []credentials.Credentials{
		{validServerAddress, "foo", "bar"},
		{validServerAddress2, "<token>", "abcd1234"},
	}

	for _, v := range valid {
		if err := Store(mockProgramFn, &v); err != nil {
			t.Fatal(err)
		}
	}

	invalid := []credentials.Credentials{
		{invalidServerAddress, "foo", "bar"},
	}

	for _, v := range invalid {
		if err := Store(mockProgramFn, &v); err == nil {
			t.Fatalf("Expected error for server %s, got nil", v.ServerURL)
		}
	}
}

func ExampleGet() {
	p := NewShellProgramFunc("docker-credential-secretservice")

	creds, err := Get(p, "https://example.com")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Got credentials for user `%s` in `%s`\n", creds.Username, creds.ServerURL)
}

func TestGet(t *testing.T) {
	valid := []credentials.Credentials{
		{validServerAddress, "foo", "bar"},
		{validServerAddress2, "<token>", "abcd1234"},
	}

	for _, v := range valid {
		c, err := Get(mockProgramFn, v.ServerURL)
		if err != nil {
			t.Fatal(err)
		}

		if c.Username != v.Username {
			t.Fatalf("expected username `%s`, got %s", v.Username, c.Username)
		}
		if c.Secret != v.Secret {
			t.Fatalf("expected secret `%s`, got %s", v.Secret, c.Secret)
		}
	}

	missingServerURLErr := credentials.NewErrCredentialsMissingServerURL()

	invalid := []struct {
		serverURL string
		err       string
	}{
		{missingCredsAddress, credentials.NewErrCredentialsNotFound().Error()},
		{invalidServerAddress, "error getting credentials - err: exited 1, out: `program failed`"},
		{"", fmt.Sprintf("error getting credentials - err: %s, out: `%s`",
			missingServerURLErr.Error(), missingServerURLErr.Error())},
	}

	for _, v := range invalid {
		_, err := Get(mockProgramFn, v.serverURL)
		if err == nil {
			t.Fatalf("Expected error for server %s, got nil", v.serverURL)
		}
		if err.Error() != v.err {
			t.Fatalf("Expected error `%s`, got `%v`", v.err, err)
		}
	}
}

func ExampleErase() {
	p := NewShellProgramFunc("docker-credential-secretservice")

	if err := Erase(p, "https://example.com"); err != nil {
		fmt.Println(err)
	}
}

func TestErase(t *testing.T) {
	if err := Erase(mockProgramFn, validServerAddress); err != nil {
		t.Fatal(err)
	}

	if err := Erase(mockProgramFn, invalidServerAddress); err == nil {
		t.Fatalf("Expected error for server %s, got nil", invalidServerAddress)
	}
}

func TestList(t *testing.T) {
	auths, err := List(mockProgramFn)
	if err != nil {
		t.Fatal(err)
	}

	if username, exists := auths[validServerAddress]; !exists || username != validUsername {
		t.Fatalf("auths[%s] returned %s, %t; expected %s, %t", validServerAddress, username, exists, validUsername, true)
	}
}
