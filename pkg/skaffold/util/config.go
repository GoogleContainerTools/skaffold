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
	"errors"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
)

// ReadConfiguration reads a `skaffold.yaml` configuration and
// returns its content.
func ReadConfiguration(filename string) ([]byte, error) {
	switch {
	case filename == "":
		return nil, errors.New("filename not specified")
	case filename == "-":
		return ioutil.ReadAll(os.Stdin)
	case IsURL(filename):
		return Download(filename)
	default:
		contents, err := ioutil.ReadFile(filename)
		if err != nil {
			// If the config file is the default `skaffold.yaml`,
			// then we also try to read `skaffold.yml`.
			if filename == "skaffold.yaml" {
				logrus.Infof("Could not open skaffold.yaml: \"%s\"", err)
				logrus.Infof("Trying to read from skaffold.yml instead")
				contents, errIgnored := ioutil.ReadFile("skaffold.yml")
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
