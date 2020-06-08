package project

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

type License struct {
	Type string `toml:"type"`
	URI  string `toml:"uri"`
}

type Project struct {
	Name     string    `toml:"name"`
	Licenses []License `toml:"licenses"`
}

type Descriptor struct {
	Project  Project                `toml:"project"`
	Build    Build                  `toml:"build"`
	Metadata map[string]interface{} `toml:"metadata"`
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
	if len(p.Project.Licenses) > 0 {
		for _, license := range p.Project.Licenses {
			if license.Type == "" && license.URI == "" {
				return errors.New("project.toml: must have a type or uri defined for each license")
			}
		}
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
