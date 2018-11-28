package config

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/docker/docker/pkg/homedir"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
)

func setupConfigDir(t *testing.T) (string, func()) {
	tmpdir, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	oldDir := Dir()
	SetDir(tmpdir)

	return tmpdir, func() {
		SetDir(oldDir)
		os.RemoveAll(tmpdir)
	}
}

func TestEmptyConfigDir(t *testing.T) {
	tmpHome, cleanup := setupConfigDir(t)
	defer cleanup()

	config, err := Load("")
	assert.NilError(t, err)

	expectedConfigFilename := filepath.Join(tmpHome, ConfigFileName)
	assert.Check(t, is.Equal(expectedConfigFilename, config.Filename))

	// Now save it and make sure it shows up in new form
	saveConfigAndValidateNewFormat(t, config, tmpHome)
}

func TestMissingFile(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	// Now save it and make sure it shows up in new form
	saveConfigAndValidateNewFormat(t, config, tmpHome)
}

func TestSaveFileToDirs(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	tmpHome += "/.docker"

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	// Now save it and make sure it shows up in new form
	saveConfigAndValidateNewFormat(t, config, tmpHome)
}

func TestEmptyFile(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	fn := filepath.Join(tmpHome, ConfigFileName)
	err = ioutil.WriteFile(fn, []byte(""), 0600)
	assert.NilError(t, err)

	_, err = Load(tmpHome)
	assert.Equal(t, errors.Cause(err), io.EOF)
	assert.ErrorContains(t, err, ConfigFileName)
}

func TestEmptyJSON(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	fn := filepath.Join(tmpHome, ConfigFileName)
	err = ioutil.WriteFile(fn, []byte("{}"), 0600)
	assert.NilError(t, err)

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	// Now save it and make sure it shows up in new form
	saveConfigAndValidateNewFormat(t, config, tmpHome)
}

func TestOldInvalidsAuth(t *testing.T) {
	invalids := map[string]string{
		`username = test`: "The Auth config file is empty",
		`username
password`: "Invalid Auth config file",
		`username = test
email`: "Invalid auth configuration file",
	}

	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	homeKey := homedir.Key()
	homeVal := homedir.Get()

	defer func() { os.Setenv(homeKey, homeVal) }()
	os.Setenv(homeKey, tmpHome)

	for content, expectedError := range invalids {
		fn := filepath.Join(tmpHome, oldConfigfile)
		err := ioutil.WriteFile(fn, []byte(content), 0600)
		assert.NilError(t, err)

		_, err = Load(tmpHome)
		assert.ErrorContains(t, err, expectedError)
	}
}

func TestOldValidAuth(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	homeKey := homedir.Key()
	homeVal := homedir.Get()

	defer func() { os.Setenv(homeKey, homeVal) }()
	os.Setenv(homeKey, tmpHome)

	fn := filepath.Join(tmpHome, oldConfigfile)
	js := `username = am9lam9lOmhlbGxv
	email = user@example.com`
	err = ioutil.WriteFile(fn, []byte(js), 0600)
	assert.NilError(t, err)

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	// defaultIndexserver is https://index.docker.io/v1/
	ac := config.AuthConfigs["https://index.docker.io/v1/"]
	if ac.Username != "joejoe" || ac.Password != "hello" {
		t.Fatalf("Missing data from parsing:\n%q", config)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)

	expConfStr := `{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "am9lam9lOmhlbGxv"
		}
	}
}`

	assert.Check(t, is.Equal(expConfStr, configStr))
}

func TestOldJSONInvalid(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	homeKey := homedir.Key()
	homeVal := homedir.Get()

	defer func() { os.Setenv(homeKey, homeVal) }()
	os.Setenv(homeKey, tmpHome)

	fn := filepath.Join(tmpHome, oldConfigfile)
	js := `{"https://index.docker.io/v1/":{"auth":"test","email":"user@example.com"}}`
	if err := ioutil.WriteFile(fn, []byte(js), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	// Use Contains instead of == since the file name will change each time
	if err == nil || !strings.Contains(err.Error(), "Invalid auth configuration file") {
		t.Fatalf("Expected an error got : %v, %v", config, err)
	}
}

func TestOldJSON(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	homeKey := homedir.Key()
	homeVal := homedir.Get()

	defer func() { os.Setenv(homeKey, homeVal) }()
	os.Setenv(homeKey, tmpHome)

	fn := filepath.Join(tmpHome, oldConfigfile)
	js := `{"https://index.docker.io/v1/":{"auth":"am9lam9lOmhlbGxv","email":"user@example.com"}}`
	if err := ioutil.WriteFile(fn, []byte(js), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	ac := config.AuthConfigs["https://index.docker.io/v1/"]
	if ac.Username != "joejoe" || ac.Password != "hello" {
		t.Fatalf("Missing data from parsing:\n%q", config)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)

	expConfStr := `{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "am9lam9lOmhlbGxv",
			"email": "user@example.com"
		}
	}
}`

	if configStr != expConfStr {
		t.Fatalf("Should have save in new form: \n'%s'\n not \n'%s'\n", configStr, expConfStr)
	}
}

func TestNewJSON(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := ` { "auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv" } } }`
	if err := ioutil.WriteFile(fn, []byte(js), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	ac := config.AuthConfigs["https://index.docker.io/v1/"]
	if ac.Username != "joejoe" || ac.Password != "hello" {
		t.Fatalf("Missing data from parsing:\n%q", config)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)

	expConfStr := `{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "am9lam9lOmhlbGxv"
		}
	}
}`

	if configStr != expConfStr {
		t.Fatalf("Should have save in new form: \n%s\n not \n%s", configStr, expConfStr)
	}
}

func TestNewJSONNoEmail(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := ` { "auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv" } } }`
	if err := ioutil.WriteFile(fn, []byte(js), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	ac := config.AuthConfigs["https://index.docker.io/v1/"]
	if ac.Username != "joejoe" || ac.Password != "hello" {
		t.Fatalf("Missing data from parsing:\n%q", config)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)

	expConfStr := `{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "am9lam9lOmhlbGxv"
		}
	}
}`

	if configStr != expConfStr {
		t.Fatalf("Should have save in new form: \n%s\n not \n%s", configStr, expConfStr)
	}
}

func TestJSONWithPsFormat(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } },
		"psFormat": "table {{.ID}}\\t{{.Label \"com.docker.label.cpu\"}}"
}`
	if err := ioutil.WriteFile(fn, []byte(js), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	if config.PsFormat != `table {{.ID}}\t{{.Label "com.docker.label.cpu"}}` {
		t.Fatalf("Unknown ps format: %s\n", config.PsFormat)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)
	if !strings.Contains(configStr, `"psFormat":`) ||
		!strings.Contains(configStr, "{{.ID}}") {
		t.Fatalf("Should have save in new form: %s", configStr)
	}
}

func TestJSONWithCredentialStore(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } },
		"credsStore": "crazy-secure-storage"
}`
	if err := ioutil.WriteFile(fn, []byte(js), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	if config.CredentialsStore != "crazy-secure-storage" {
		t.Fatalf("Unknown credential store: %s\n", config.CredentialsStore)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)
	if !strings.Contains(configStr, `"credsStore":`) ||
		!strings.Contains(configStr, "crazy-secure-storage") {
		t.Fatalf("Should have save in new form: %s", configStr)
	}
}

func TestJSONWithCredentialHelpers(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	fn := filepath.Join(tmpHome, ConfigFileName)
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } },
		"credHelpers": { "images.io": "images-io", "containers.com": "crazy-secure-storage" }
}`
	if err := ioutil.WriteFile(fn, []byte(js), 0600); err != nil {
		t.Fatal(err)
	}

	config, err := Load(tmpHome)
	assert.NilError(t, err)

	if config.CredentialHelpers == nil {
		t.Fatal("config.CredentialHelpers was nil")
	} else if config.CredentialHelpers["images.io"] != "images-io" ||
		config.CredentialHelpers["containers.com"] != "crazy-secure-storage" {
		t.Fatalf("Credential helpers not deserialized properly: %v\n", config.CredentialHelpers)
	}

	// Now save it and make sure it shows up in new form
	configStr := saveConfigAndValidateNewFormat(t, config, tmpHome)
	if !strings.Contains(configStr, `"credHelpers":`) ||
		!strings.Contains(configStr, "images.io") ||
		!strings.Contains(configStr, "images-io") ||
		!strings.Contains(configStr, "containers.com") ||
		!strings.Contains(configStr, "crazy-secure-storage") {
		t.Fatalf("Should have save in new form: %s", configStr)
	}
}

// Save it and make sure it shows up in new form
func saveConfigAndValidateNewFormat(t *testing.T, config *configfile.ConfigFile, configDir string) string {
	assert.NilError(t, config.Save())

	buf, err := ioutil.ReadFile(filepath.Join(configDir, ConfigFileName))
	assert.NilError(t, err)
	assert.Check(t, is.Contains(string(buf), `"auths":`))
	return string(buf)
}

func TestConfigDir(t *testing.T) {
	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	if Dir() == tmpHome {
		t.Fatalf("Expected ConfigDir to be different than %s by default, but was the same", tmpHome)
	}

	// Update configDir
	SetDir(tmpHome)

	if Dir() != tmpHome {
		t.Fatalf("Expected ConfigDir to %s, but was %s", tmpHome, Dir())
	}
}

func TestJSONReaderNoFile(t *testing.T) {
	js := ` { "auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } } }`

	config, err := LoadFromReader(strings.NewReader(js))
	assert.NilError(t, err)

	ac := config.AuthConfigs["https://index.docker.io/v1/"]
	if ac.Username != "joejoe" || ac.Password != "hello" {
		t.Fatalf("Missing data from parsing:\n%q", config)
	}

}

func TestOldJSONReaderNoFile(t *testing.T) {
	js := `{"https://index.docker.io/v1/":{"auth":"am9lam9lOmhlbGxv","email":"user@example.com"}}`

	config, err := LegacyLoadFromReader(strings.NewReader(js))
	assert.NilError(t, err)

	ac := config.AuthConfigs["https://index.docker.io/v1/"]
	if ac.Username != "joejoe" || ac.Password != "hello" {
		t.Fatalf("Missing data from parsing:\n%q", config)
	}
}

func TestJSONWithPsFormatNoFile(t *testing.T) {
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv", "email": "user@example.com" } },
		"psFormat": "table {{.ID}}\\t{{.Label \"com.docker.label.cpu\"}}"
}`
	config, err := LoadFromReader(strings.NewReader(js))
	assert.NilError(t, err)

	if config.PsFormat != `table {{.ID}}\t{{.Label "com.docker.label.cpu"}}` {
		t.Fatalf("Unknown ps format: %s\n", config.PsFormat)
	}

}

func TestJSONSaveWithNoFile(t *testing.T) {
	js := `{
		"auths": { "https://index.docker.io/v1/": { "auth": "am9lam9lOmhlbGxv" } },
		"psFormat": "table {{.ID}}\\t{{.Label \"com.docker.label.cpu\"}}"
}`
	config, err := LoadFromReader(strings.NewReader(js))
	assert.NilError(t, err)
	err = config.Save()
	assert.ErrorContains(t, err, "with empty filename")

	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	fn := filepath.Join(tmpHome, ConfigFileName)
	f, _ := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()

	assert.NilError(t, config.SaveToWriter(f))
	buf, err := ioutil.ReadFile(filepath.Join(tmpHome, ConfigFileName))
	assert.NilError(t, err)
	expConfStr := `{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "am9lam9lOmhlbGxv"
		}
	},
	"psFormat": "table {{.ID}}\\t{{.Label \"com.docker.label.cpu\"}}"
}`
	if string(buf) != expConfStr {
		t.Fatalf("Should have save in new form: \n%s\nnot \n%s", string(buf), expConfStr)
	}
}

func TestLegacyJSONSaveWithNoFile(t *testing.T) {
	js := `{"https://index.docker.io/v1/":{"auth":"am9lam9lOmhlbGxv","email":"user@example.com"}}`
	config, err := LegacyLoadFromReader(strings.NewReader(js))
	assert.NilError(t, err)
	err = config.Save()
	assert.ErrorContains(t, err, "with empty filename")

	tmpHome, err := ioutil.TempDir("", "config-test")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpHome)

	fn := filepath.Join(tmpHome, ConfigFileName)
	f, _ := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()

	assert.NilError(t, config.SaveToWriter(f))
	buf, err := ioutil.ReadFile(filepath.Join(tmpHome, ConfigFileName))
	assert.NilError(t, err)

	expConfStr := `{
	"auths": {
		"https://index.docker.io/v1/": {
			"auth": "am9lam9lOmhlbGxv",
			"email": "user@example.com"
		}
	}
}`

	if string(buf) != expConfStr {
		t.Fatalf("Should have save in new form: \n%s\n not \n%s", string(buf), expConfStr)
	}
}

func TestLoadDefaultConfigFile(t *testing.T) {
	dir, cleanup := setupConfigDir(t)
	defer cleanup()
	buffer := new(bytes.Buffer)

	filename := filepath.Join(dir, ConfigFileName)
	content := []byte(`{"PsFormat": "format"}`)
	err := ioutil.WriteFile(filename, content, 0644)
	assert.NilError(t, err)

	configFile := LoadDefaultConfigFile(buffer)
	credStore := credentials.DetectDefaultStore("")
	expected := configfile.New(filename)
	expected.CredentialsStore = credStore
	expected.PsFormat = "format"

	assert.Check(t, is.DeepEqual(expected, configFile))
}
