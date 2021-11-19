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

package util

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

var stdin []byte

// Fs is the underlying filesystem to use for reading skaffold project files & configuration.  OS FS by default
var Fs = afero.NewOsFs()

// ReadConfiguration reads a `skaffold.yaml` configuration and
// returns its content.
func ReadConfiguration(filename string) ([]byte, error) {
	switch {
	case filename == "":
		return nil, errors.New("filename not specified")
	case filename == "-":
		if len(stdin) == 0 {
			var err error
			stdin, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				return []byte{}, err
			}
		}
		return stdin, nil
	case IsURL(filename):
		return Download(filename)
	default:
		fp := filename
		if !filepath.IsAbs(fp) {
			dir, err := os.Getwd()
			if err != nil {
				return []byte{}, err
			}
			fp = filepath.Join(dir, fp)
		}
		contents, err := afero.ReadFile(Fs, fp)
		if err != nil {
			// If the config file is the default `skaffold.yaml`,
			// then we also try to read `skaffold.yml`.
			if filename == "skaffold.yaml" {
				log.Entry(context.TODO()).Infof("Could not open skaffold.yaml: \"%s\"", err)
				log.Entry(context.TODO()).Info("Trying to read from skaffold.yml instead")
				contents, errIgnored := afero.ReadFile(Fs, filepath.Join(filepath.Dir(fp), "skaffold.yml"))
				if errIgnored != nil {
					// Return original error because it's the one that matters
					return nil, err
				}

				return contents, nil
			}
		}

		return contents, err
	}
}

func ReadFile(filename string) ([]byte, error) {
	if !filepath.IsAbs(filename) {
		dir, err := os.Getwd()
		if err != nil {
			return []byte{}, err
		}
		filename = filepath.Join(dir, filename)
	}
	return afero.ReadFile(Fs, filename)
}
