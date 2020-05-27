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

// This whole file is copy/pasted from
// https://github.com/buildpacks/pack/blob/master/internal/project/project.go
package buildpacks

import (
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

type Buildpack struct {
	ID      string `toml:"id"`
	Version string `toml:"version"`
	URI     string `toml:"uri"`
}

type EnvVar struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}

type Build struct {
	Include    []string    `toml:"include"`
	Exclude    []string    `toml:"exclude"`
	Buildpacks []Buildpack `toml:"buildpacks"`
	Env        []EnvVar    `toml:"env"`
}

type Descriptor struct {
	Build Build `toml:"build"`
}

func ReadProjectDescriptor(pathToFile string) (Descriptor, error) {
	projectTomlContents, err := ioutil.ReadFile(pathToFile)
	if err != nil {
		return Descriptor{}, err
	}

	var descriptor Descriptor
	_, err = toml.Decode(string(projectTomlContents), &descriptor)
	if err != nil {
		return Descriptor{}, err
	}

	return descriptor, descriptor.validate()
}

func (p Descriptor) validate() error {
	if p.Build.Exclude != nil && p.Build.Include != nil {
		return errors.New("project.toml: cannot have both include and exclude defined")
	}

	for _, bp := range p.Build.Buildpacks {
		if bp.ID == "" && bp.URI == "" {
			return errors.New("project.toml: buildpacks must have an id or url defined")
		}
		if bp.URI != "" && bp.Version != "" {
			return errors.New("project.toml: buildpacks cannot have both uri and version defined")
		}
	}

	return nil
}
