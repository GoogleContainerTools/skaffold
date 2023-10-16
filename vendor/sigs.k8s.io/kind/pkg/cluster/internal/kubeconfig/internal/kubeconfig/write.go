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
	"os"
	"path/filepath"

	"sigs.k8s.io/kind/pkg/errors"
)

// write writes cfg to configPath
// it will ensure the directories in the path if necessary
func write(cfg *Config, configPath string) error {
	encoded, err := Encode(cfg)
	if err != nil {
		return err
	}
	// NOTE: 0755 / 0600 are to match client-go
	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrap(err, "failed to create directory for KUBECONFIG")
		}
	}
	if err := os.WriteFile(configPath, encoded, 0600); err != nil {
		return errors.Wrap(err, "failed to write KUBECONFIG")
	}
	return nil
}
