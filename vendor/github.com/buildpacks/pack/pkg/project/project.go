package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"

	"github.com/buildpacks/pack/pkg/logging"
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

var parsers = map[string]func(string) (types.Descriptor, toml.MetaData, error){
	"0.1": v01.NewDescriptor,
	"0.2": v02.NewDescriptor,
}

func ReadProjectDescriptor(pathToFile string, logger logging.Logger) (types.Descriptor, error) {
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
		logger.Warn("No schema version declared in project.toml, defaulting to schema version 0.1")
		version = "0.1"
	}

	if _, ok := parsers[version]; !ok {
		return types.Descriptor{}, fmt.Errorf("unknown project descriptor schema version %s", version)
	}

	descriptor, tomlMetaData, err := parsers[version](string(projectTomlContents))
	if err != nil {
		return types.Descriptor{}, err
	}

	warnIfTomlContainsKeysNotSupportedBySchema(version, tomlMetaData, logger)

	return descriptor, validate(descriptor)
}

func warnIfTomlContainsKeysNotSupportedBySchema(schemaVersion string, tomlMetaData toml.MetaData, logger logging.Logger) {
	unsupportedKeys := []string{}

	for _, undecoded := range tomlMetaData.Undecoded() {
		keyName := undecoded.String()
		if unsupportedKey(keyName, schemaVersion) {
			unsupportedKeys = append(unsupportedKeys, keyName)
		}
	}

	if len(unsupportedKeys) != 0 {
		logger.Warnf("The following keys declared in project.toml are not supported in schema version %s:\n", schemaVersion)
		for _, unsupported := range unsupportedKeys {
			logger.Warnf("- %s\n", unsupported)
		}
		logger.Warn("The above keys will be ignored. If this is not intentional, try updating your schema version.\n")
	}
}

func unsupportedKey(keyName, schemaVersion string) bool {
	switch schemaVersion {
	case "0.1":
		// filter out any keys from [metadata] and any other custom table defined by end-users
		return strings.HasPrefix(keyName, "project.") || strings.HasPrefix(keyName, "build.") || strings.Contains(keyName, "io.buildpacks")
	case "0.2":
		// filter out any keys from [_.metadata] and any other custom table defined by end-users
		return strings.Contains(keyName, "io.buildpacks") || (strings.HasPrefix(keyName, "_.") && !strings.HasPrefix(keyName, "_.metadata"))
	}
	return true
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
