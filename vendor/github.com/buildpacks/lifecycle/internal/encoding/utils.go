package encoding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// json

// ToJSONMaybe returns the provided interface as JSON if marshaling is successful,
// or as a string if an error is encountered.
// It is only intended to be used for logging.
func ToJSONMaybe(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%s", v) // hopefully v is a Stringer
	}
	return string(b)
}

// WriteJSON writes data as JSON to the specified path, creating parent directories as needed.
func WriteJSON(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(data)
}

// toml

// MarshalTOML encodes v as TOML and returns the resulting bytes.
func MarshalTOML(v any) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteTOML writes data as TOML to the specified path, creating parent directories as needed.
func WriteTOML(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(data)
}
