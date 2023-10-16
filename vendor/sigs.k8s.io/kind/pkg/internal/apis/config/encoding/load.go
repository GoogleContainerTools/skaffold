/*
Copyright 2018 The Kubernetes Authors.

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

package encoding

import (
	"bytes"
	"os"

	yaml "gopkg.in/yaml.v3"

	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/errors"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// Load reads the file at path and attempts to convert into a `kind` Config; the file
// can be one of the different API versions defined in scheme.
// If path == "" then the default config is returned
// If path == "-" then reads from stdin
func Load(path string) (*config.Cluster, error) {
	// special case: empty path -> default config
	// TODO(bentheelder): consider removing this
	if path == "" {
		out := &config.Cluster{}
		config.SetDefaultsCluster(out)
		return out, nil
	}

	// read in file
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "error reading file")
	}

	return Parse(raw)
}

// Parse parses a cluster config from raw (yaml) bytes
// It will always return the current internal version after defaulting and
// conversion from the read version
func Parse(raw []byte) (*config.Cluster, error) {
	// get kind & apiVersion
	tm := typeMeta{}
	if err := yaml.Unmarshal(raw, &tm); err != nil {
		return nil, errors.Wrap(err, "could not determine kind / apiVersion for config")
	}

	// decode specific (apiVersion, kind)
	switch tm.APIVersion {
	// handle v1alpha4
	case "kind.x-k8s.io/v1alpha4":
		if tm.Kind != "Cluster" {
			return nil, errors.Errorf("unknown kind %s for apiVersion: %s", tm.Kind, tm.APIVersion)
		}
		// load version
		cfg := &v1alpha4.Cluster{}
		if err := yamlUnmarshalStrict(raw, cfg); err != nil {
			return nil, errors.Wrap(err, "unable to decode config")
		}
		// apply defaults for version and convert
		return V1Alpha4ToInternal(cfg), nil
	}

	// unknown apiVersion if we haven't already returned ...
	return nil, errors.Errorf("unknown apiVersion: %s", tm.APIVersion)
}

// basically metav1.TypeMeta, but with yaml tags
type typeMeta struct {
	Kind       string `yaml:"kind,omitempty"`
	APIVersion string `yaml:"apiVersion,omitempty"`
}

func yamlUnmarshalStrict(raw []byte, v interface{}) error {
	d := yaml.NewDecoder(bytes.NewReader(raw))
	d.KnownFields(true)
	return d.Decode(v)
}
