package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/pkg/project/types"
	v01 "github.com/buildpacks/pack/pkg/project/v01"
	v02 "github.com/buildpacks/pack/pkg/project/v02"
)

type Project struct {
	Version string `toml:"schema-version"`
}

type VersionDescriptor struct {
	Project Project `toml:"_"`
}

var parsers = map[string]func(string) (types.Descriptor, error){
	"0.1": v01.NewDescriptor,
	"0.2": v02.NewDescriptor,
}

func ReadProjectDescriptor(pathToFile string) (types.Descriptor, error) {
	projectTomlContents, err := os.ReadFile(filepath.Clean(pathToFile))
	if err != nil {
		return types.Descriptor{}, err
	}

	var versionDescriptor struct {
		Project struct {
			Version string `toml:"schema-version"`
		} `toml:"_"`
	}

	_, err = toml.Decode(string(projectTomlContents), &versionDescriptor)
	if err != nil {
		return types.Descriptor{}, errors.Wrapf(err, "parsing schema version")
	}

	version := versionDescriptor.Project.Version
	if version == "" {
		version = "0.1"
	}

	if _, ok := parsers[version]; !ok {
		return types.Descriptor{}, fmt.Errorf("unknown project descriptor schema version %s", version)
	}

	descriptor, err := parsers[version](string(projectTomlContents))
	if err != nil {
		return types.Descriptor{}, err
	}

	return descriptor, validate(descriptor)
}

func validate(p types.Descriptor) error {
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
