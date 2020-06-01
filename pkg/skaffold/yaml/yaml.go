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

package yamlutil

import (
	"bytes"
	"io"

	yaml "gopkg.in/yaml.v3"
)

func UnmarshalStrict(in []byte, out interface{}) error {
	return unmarshall(in, out, true)
}

func Unmarshal(in []byte, out interface{}) error {
	return unmarshall(in, out, false)
}

func Marshal(in interface{}) (out []byte, err error) {
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(in); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func unmarshall(in []byte, out interface{}, strict bool) error {
	b := bytes.NewReader(in)
	decoder := yaml.NewDecoder(b)
	decoder.KnownFields(strict)
	if err := decoder.Decode(out); err != nil {
		// yamlv3.Unmarshal swallows EOF to return empty object for empty string.
		if err != io.EOF {
			return err
		}
	}
	return nil
}
