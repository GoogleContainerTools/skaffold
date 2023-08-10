/*
Copyright 2019 The Kubernetes Authors.

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

package kubeconfig

import (
	"bytes"

	yaml "gopkg.in/yaml.v3"
	kubeyaml "sigs.k8s.io/yaml"

	"sigs.k8s.io/kind/pkg/errors"
)

// Encode encodes the cfg to yaml
func Encode(cfg *Config) ([]byte, error) {
	// NOTE: kubernetes's yaml library doesn't handle inline fields very well
	// so we're not using that to marshal
	encoded, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode KUBECONFIG")
	}

	// normalize with kubernetes's yaml library
	// this is not strictly necessary, but it ensures minimal diffs when
	// modifying kubeconfig files, which is nice to have
	encoded, err = normYaml(encoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to normalize KUBECONFIG encoding")
	}

	return encoded, nil
}

// normYaml round trips yaml bytes through sigs.k8s.io/yaml to normalize them
// versus other kubernetes ecosystem yaml output
func normYaml(y []byte) ([]byte, error) {
	var unstructured interface{}
	if err := kubeyaml.Unmarshal(y, &unstructured); err != nil {
		return nil, err
	}
	encoded, err := kubeyaml.Marshal(&unstructured)
	if err != nil {
		return nil, err
	}
	// special case: don't write anything when empty
	if bytes.Equal(encoded, []byte("{}\n")) {
		return []byte{}, nil
	}
	return encoded, nil
}
