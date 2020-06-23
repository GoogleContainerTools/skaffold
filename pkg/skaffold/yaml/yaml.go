/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package yaml

import (
	"bytes"
	"io"

	yaml "gopkg.in/yaml.v3"
)

// UnmarshalStrict is like Unmarshal except that any fields that are found
// in the data that do not have corresponding struct members, or mapping
// keys that are duplicates, will result in an error.
// This is ensured by setting `KnownFields` true on `yaml.Decoder`.
func UnmarshalStrict(in []byte, out interface{}) error {
	b := bytes.NewReader(in)
	decoder := yaml.NewDecoder(b)
	decoder.KnownFields(true)
	// Ignore io.EOF which signals expected end of input.
	// This happens when input stream is empty or nil.
	if err := decoder.Decode(out); err != io.EOF {
		return err
	}
	return nil
}

// Unmarshal is wrapper around yaml.Unmarshal
func Unmarshal(in []byte, out interface{}) error {
	return yaml.Unmarshal(in, out)
}

// Marshal is same as yaml.Marshal except it creates a `yaml.Encoder` with
// indent space 2 for encoding.
func Marshal(in interface{}) (out []byte, err error) {
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(in); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
