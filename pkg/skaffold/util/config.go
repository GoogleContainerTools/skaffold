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
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// ConfigFile represents a `skaffold.yaml` configuration file
type ConfigFile interface {
	Read() ([]byte, error)
	Dir() string
}

type urlConfig struct {
	url string
}

func (c *urlConfig) Read() ([]byte, error) {
	return Download(c.url)
}

func (c *urlConfig) Dir() string {
	wd, _ := os.Getwd()
	return wd
}

type fileConfig struct {
	path string
}

func (c *fileConfig) Read() ([]byte, error) {
	return ioutil.ReadFile(c.path)
}

func (c *fileConfig) Dir() string {
	return filepath.Dir(c.path)
}

type stdinConfig struct{}

func (c *stdinConfig) Read() ([]byte, error) {
	return ioutil.ReadAll(os.Stdin)
}

func (c *stdinConfig) Dir() string {
	wd, _ := os.Getwd()
	return wd
}

func NewConfigFile(filename string) (ConfigFile, error) {
	switch {
	case filename == "":
		return nil, errors.New("filename not specified")
	case IsURL(filename):
		return &urlConfig{url: filename}, nil
	case filename == "-":
		return &stdinConfig{}, nil
	default:
		if _, err := os.Stat(filename); err != nil {
			// If the config file is the default `skaffold.yaml`,
			// then we also try to read `skaffold.yml`.
			if filepath.Base(filename) == "skaffold.yaml" {
				logrus.Infof("Could not open skaffold.yaml: \"%s\"", err)
				logrus.Infof("Trying to read from skaffold.yml instead")
				filename = filepath.Join(filepath.Dir(filename), "skaffold.yml")
				if _, err := os.Stat(filename); err != nil {
					return nil, err
				}
				return &fileConfig{path: filename}, nil
			}
			return nil, err
		}

		return &fileConfig{path: filename}, nil
	}
}
